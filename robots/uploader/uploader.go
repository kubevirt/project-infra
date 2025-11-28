package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"kubevirt.io/project-infra/pkg/mirror"

	"cloud.google.com/go/storage"
	"github.com/bazelbuild/buildtools/build"
	"google.golang.org/api/option"
)

var targetMirrorURLPattern = regexp.MustCompile(`^https://storage.googleapis.com/.+`)

type options struct {
	dryRun          bool
	bucket          string
	workspacePath   string
	continueOnError bool
	verify          bool
	dir             string
	requiresAuth    bool
}

func (o *options) Validate() error {
	if o.workspacePath == "" {
		return fmt.Errorf("Path to the workspace file is required")
	}
	return nil
}

func gatherOptions() options {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.BoolVar(&o.dryRun, "dry-run", true, "Dry run for testing. Uses API tokens but does not mutate.")
	fs.BoolVar(&o.verify, "verify", false, "Verify that all artifacts are uploaded and that they have the right shasum")
	fs.BoolVar(&o.continueOnError, "continue-on-error", false, "Try to upload as many artifacts as possible. Exit code will still be non-zero in case of errors")
	fs.StringVar(&o.bucket, "bucket", "builddeps", "bucket where to upload")
	fs.StringVar(&o.dir, "dir", "", "directory inside the bucket")
	fs.BoolVar(&o.requiresAuth, "requires-auth", false, "set to true if the bucket requires authentication for downloading artifacts")
	fs.StringVar(&o.workspacePath, "workspace", "", "path to the workspace file")
	fs.Parse(os.Args[1:])
	return o
}

func main() {
	options := gatherOptions()
	if err := options.Validate(); err != nil {
		log.Fatalf("invalid arguments: %v", err)
	}
	fmt.Println(options.dryRun)
	rawFile, err := os.ReadFile(options.workspacePath)
	if err != nil {
		log.Fatalf("could not read workspace file: %v", err)
	}
	workspace, err := build.ParseWorkspace("workspace", rawFile)
	if err != nil {
		log.Fatalf("could not load workspace file: %v", err)
	}
	artifacts, err := mirror.GetArtifacts(workspace)
	if err != nil {
		log.Fatalf("could not read artifacts: %v", err)
	}

	if options.verify {
		verify(options, artifacts)
	} else {
		upload(options, workspace, artifacts)
	}
}

func verify(options options, artifacts []mirror.Artifact) {
	ctx := context.Background()
	var opts []option.ClientOption
	if options.dryRun {
		opts = append(opts, option.WithoutAuthentication())
	}
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		log.Fatalf("Failed to create new storage client: %v.\n", err)
	}
	failed := false
	for _, artifact := range artifacts {
		newFileUrl := mirror.GenerateFilePath(options.bucket, options.dir, &artifact)
		err := mirror.VerifyArtifact(ctx, client, artifact, options.dir, options.bucket)
		if err != nil {
			log.Printf("failed to upload %s to %s: %s", artifact.Name(), newFileUrl, err)
			failed = true
		}
	}

	if failed {
		os.Exit(1)
	}
}

func upload(options options, workspace *build.File, artifacts []mirror.Artifact) {
	ctx := context.Background()
	var opts []option.ClientOption
	if options.dryRun {
		opts = append(opts, option.WithoutAuthentication())
	}
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		log.Fatalf("Failed to create new storage client: %v.\n", err)
	}
	invalid := mirror.FilterArtifactsWithoutMirror(artifacts, targetMirrorURLPattern)

	failed := false
	var authPatterns []*build.KeyValueExpr
	if options.requiresAuth {
		authPatterns = append(authPatterns, &build.KeyValueExpr{
			Key:   &build.StringExpr{Value: "storage.googleapis.com"},
			Value: &build.StringExpr{Value: "Bearer <password>"},
		})
	}
	for _, artifact := range invalid {
		newFileUrl := mirror.GenerateFilePath(options.bucket, options.dir, &artifact)
		err := mirror.WriteToBucket(options.dryRun, ctx, client, artifact, options.bucket, options.dir, http.DefaultClient)
		if err != nil {
			log.Printf("failed to upload %s to %s: %s", artifact.Name(), newFileUrl, err)
			if options.continueOnError {
				failed = true
			} else {
				os.Exit(1)
			}
		}
		artifact.AppendURL(newFileUrl)
		if len(authPatterns) > 0 {
			artifact.SetAuthPatterns(authPatterns)
		}
	}

	mirror.RemoveStaleDownloadURLS(artifacts, targetMirrorURLPattern, http.DefaultClient)
	err = mirror.CheckArtifactsHaveURLS(artifacts)
	if err != nil {
		log.Fatalf("could not update workspace items: %v", err)
	}

	err = mirror.WriteWorkspace(options.dryRun, workspace, options.workspacePath)
	if err != nil {
		log.Fatalf("could not write workspace file: %v", err)
	}

	if failed {
		os.Exit(1)
	}
}

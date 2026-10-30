package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	modulePath      string
	continueOnError bool
	verify          bool
	dir             string
	requiresAuth    bool
}

func (o *options) Validate() error {
	if o.workspacePath == "" && o.modulePath == "" {
		return fmt.Errorf("Path to the workspace or module file is required")
	}
	if o.workspacePath != "" && o.modulePath != "" {
		return fmt.Errorf("-workspace and -module are mutually exclusive")
	}
	return nil
}

func resolveBazelFile(o options) string {
	if o.modulePath != "" {
		return o.modulePath
	}
	if _, err := os.Stat(o.workspacePath); errors.Is(err, os.ErrNotExist) {
		modPath := filepath.Join(filepath.Dir(o.workspacePath), "MODULE.bazel")
		if _, modErr := os.Stat(modPath); modErr == nil {
			log.Printf("WORKSPACE %s not found, falling back to %s", o.workspacePath, modPath)
			return modPath
		}
	}
	return o.workspacePath
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
	fs.StringVar(&o.modulePath, "module", "", "path to the MODULE.bazel file")
	_ = fs.Parse(os.Args[1:])
	return o
}

func main() {
	options := gatherOptions()
	if err := options.Validate(); err != nil {
		log.Fatalf("invalid arguments: %v", err)
	}
	options.workspacePath = resolveBazelFile(options)
	fmt.Println(options.dryRun)
	workspace, err := mirror.LoadWorkspace(options.workspacePath)
	if err != nil {
		log.Fatalf("could not load workspace file: %v", err)
	}
	artifacts, err := mirror.GetArtifacts(workspace)
	if err != nil {
		log.Fatalf("could not read artifacts: %v", err)
	}
	lockPaths, err := mirror.GetLockFiles(workspace, filepath.Dir(options.workspacePath))
	if err != nil {
		log.Fatalf("could not resolve lockfiles: %v", err)
	}
	var lockFiles []*mirror.LockFile
	for _, lockPath := range lockPaths {
		lf, lockArtifacts, err := mirror.LoadLockFile(lockPath)
		if err != nil {
			log.Fatalf("could not load lockfile %s: %v", lockPath, err)
		}
		lockFiles = append(lockFiles, lf)
		artifacts = append(artifacts, lockArtifacts...)
	}

	if options.verify {
		verify(options, artifacts)
	} else {
		upload(options, workspace, lockFiles, artifacts)
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
		newFileUrl := mirror.GenerateFilePath(options.bucket, options.dir, artifact)
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

func upload(options options, workspace *build.File, lockFiles []*mirror.LockFile, artifacts []mirror.Artifact) {
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
		newFileUrl := mirror.GenerateFilePath(options.bucket, options.dir, artifact)
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
	for _, lf := range lockFiles {
		if err := lf.Write(options.dryRun); err != nil {
			log.Fatalf("could not write lockfile: %v", err)
		}
	}

	if failed {
		os.Exit(1)
	}
}

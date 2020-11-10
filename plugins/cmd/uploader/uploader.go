package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"

	"cloud.google.com/go/storage"
	"github.com/bazelbuild/buildtools/build"
	"kubevirt.io/project-infra/plugins/mirror"
)

type options struct {
	dryRun        bool
	bucket        string
	workspacePath string
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
	fs.StringVar(&o.bucket, "bucket", "builddeps", "bucket where to upload")
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

	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create new storage client: %v.\n", err)
	}

	rawFile, err := ioutil.ReadFile(options.workspacePath)
	workspace, err := build.ParseWorkspace("workspace", rawFile)
	if err != nil {
		log.Fatalf("could not load workspace file: %v", err)
	}
	artifacts, err := mirror.GetArtifacts(workspace)
	if err != nil {
		log.Fatalf("could not read artifacts: %v", err)
	}
	invalid := mirror.FilterArtifactsWithoutMirror(artifacts, regexp.MustCompile(`^https://storage.googleapis.com/.+`))

	for _, artifact := range invalid {
		newFileUrl := mirror.GenerateFilePath(options.bucket, &artifact)
		err := mirror.WriteToBucket(options.dryRun, ctx, client, artifact.URLs()[0], options.bucket, artifact.SHA256())
		if err != nil {
			log.Fatalf("failed to upload %s to %s: %s", artifact.URLs()[0], newFileUrl, err)
		}
		artifact.AppendURL(newFileUrl)
	}

	err = mirror.WriteWorkspace(options.dryRun, workspace, options.workspacePath)
	if err != nil {
		log.Fatalf("could not write workspace file: %v", err)
	}
}

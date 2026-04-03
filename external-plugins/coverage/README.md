# coverage - external plugin for prow

## Motivation

Go unit test coverage is an important metric for maintaining code quality across KubeVirt repositories. Rather than requiring each repository to define its own coverage job, the coverage plugin automates this by triggering a coverage ProwJob on any pull request that contains Go file changes.

## Overview

The coverage plugin is a Prow external plugin that:
- Listens for GitHub pull request webhooks (`opened` and `synchronize` events)
- Detects when a PR contains any `.go` file changes
- Automatically creates a coverage ProwJob for the target repository
- Runs `go test` and generates an HTML coverage report via `covreport`
- Makes the coverage report browsable via Prow's Spyglass UI through the GitHub status link

## Configuration

The plugin is registered in:
- `github/ci/prow-deploy/kustom/base/configs/current/plugins/plugins.yaml`

Images: 
- `quay.io/kubevirtci/coverage` Plugin server — runs the webhook handler in the cluster 
- `quay.io/kubevirtci/covreport` ProwJob runner — contains the Go toolchain and `covreport` tool

Deployment manifests:
- `github/ci/prow-deploy/kustom/base/manifests/local/prow-coverage-deployment.yaml`
- `github/ci/prow-deploy/kustom/base/manifests/local/prow-coverage-service.yaml`
- `github/ci/prow-deploy/kustom/base/manifests/local/prow-coverage-rbac.yaml`

The `--jobs-namespace` flag is set via a kustomize overlay patch in:
- `github/ci/prow-deploy/kustom/overlays/kubevirt-prow-control-plane/patches/JsonRFC6902/prow-coverage-deployment.yaml`

## Development

To build and push the plugin images:

```bash
# Build and push both images
make push

# Push only the plugin server image
make push-plugin

# Push only the ProwJob runner image
make push-job
```

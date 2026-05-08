# test-subset - external plugin for Prow

## Motivation

Running a full e2e test suite on a pull request can be time-consuming when only needing to verify a specific area. The test-subset plugin allows org members to trigger a targeted subset of e2e 
tests on a PR by commenting a slash command, without modifying any code or job configuration.

## Overview

Test-subset is a Prow external plugin that:
- Listens for GitHub issue comment events (`created` on open PRs) to handle `/test-subset` commands
- Parses the command to extract a target job name and test filter parameters
- Loads the named presubmit job from the Prow configuration and injects environment variables to scope the test run
- Creates a ProwJob prefixed with `test-subset-` so it appears as a separate GitHub status context and does not interfere with regular CI

The following parameters are supported, at least one must be provided:

- `--filter` — sets `KUBEVIRT_LABEL_FILTER`. Ginkgo label filter expression, auto-wrapped in parentheses if missing.
- `--focus` — sets `KUBEVIRT_E2E_FOCUS`. Ginkgo focus string.
- `--verbosity` — sets `KUBEVIRT_VERBOSITY`. Component verbosity settings.

## Usage

Comment on a PR to trigger a subset test run:

```
# Run sig-network tests with a label filter
/test-subset pull-kubevirt-e2e-k8s-1.35-sig-network --filter "SRIOV"

# Run sig-compute tests matching a focus string
/test-subset pull-kubevirt-e2e-k8s-1.35-sig-compute --focus "live migration"

# Combine multiple parameters
/test-subset pull-kubevirt-e2e-k8s-1.35-sig-compute-migrations --filter "(GPU)" --focus "live migration" --verbosity "virtLauncher:3,virtHandler:3"
```

## Authorization

A test-subset run can be triggered if the user is:
- A member of the `kubevirt` GitHub organization

## Configuration

The plugin is registered in:
- `github/ci/prow-deploy/kustom/base/configs/current/plugins/plugins.yaml`

Deployment manifests:
- `github/ci/prow-deploy/kustom/base/manifests/local/test-subset-deployment.yaml`
- `github/ci/prow-deploy/kustom/base/manifests/local/test-subset-service.yaml`
- `github/ci/prow-deploy/kustom/base/manifests/local/test-subset-rbac.yaml`

The `--jobs-namespace` flag is set via a kustomize overlay patch in:
- `github/ci/prow-deploy/kustom/overlays/kubevirt-prow-control-plane/patches/JsonRFC6902/test-subset-deployment.yaml`

## Limitations

- Only works for the `kubevirt/kubevirt` repository
- Only PRs targeting `main` or `master` branches are supported
- If validation fails (wrong repo, unauthorized user, job not found, bad parameters), no comment is posted back to the PR — errors are only written to the plugin logs
- Event handling is fully asynchronous — the webhook returns immediately with no indication of whether the job was successfully created

## Development

Build the binary:

```bash
make build
```

Run tests:

```bash
make test
```

Format and push the image:

```bash
make all
```

Push the image only:

```bash
make push
```

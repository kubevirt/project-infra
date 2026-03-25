# rehearse - external plugin for prow

## Motivation

When modifying Prow job configurations in a pull request, there is no way to
verify the changes work correctly before merging. The rehearse plugin solves
this by allowing contributors to trigger a test run of modified Prow jobs
directly from a pull request, providing pre-merge feedback without needing
to merge first.

## Overview

Rehearse is a Prow external plugin that:
- Listens for GitHub issue comment events (`created` on open PRs) to handle `/rehearse` commands
- When `--always-run` is enabled (disabled by default), also listens for pull request `opened` and `synchronize` events to automatically trigger rehearsals
- Detects changes to Prow job configuration files in a PR by rebasing and diffing the PR head against the base
- Compares job configs at the PR head vs base to identify modified or new presubmit jobs
- Creates rehearsal ProwJobs prefixed with `rehearsal-` for each modified job
- Jobs with the annotation `rehearsal.restricted: "true"` are excluded from rehearsal

## Usage

Comment on a PR to trigger rehearsals:

`/rehearse` or `/rehearse all` - Rehearse all modified jobs
`/rehearse <job-name>` - Rehearse a specific job
`/rehearse ?` - List all jobs available for rehearsal

Multiple `/rehearse <job-name>` lines can be included in a single comment to rehearse several specific jobs at once.

## Authorization

A rehearsal can be triggered if the user is:
- A **top-level approver** in `project-infra`
- A **KubeVirt org member** who is a leaf approver for all files changed in the PR (both the PR author and the commenting user must be org members)
- A **KubeVirt org member** if the PR has the `ok-to-rehearse` label (both the PR author and the commenting user must be org members)

## Configuration

The plugin is registered in:
- `github/ci/prow-deploy/kustom/base/configs/current/plugins/plugins.yaml`

Deployment manifests:
- `github/ci/prow-deploy/kustom/base/manifests/local/prow-rehearse-deployment.yaml`
- `github/ci/prow-deploy/kustom/base/manifests/local/prow-rehearse-service.yaml`
- `github/ci/prow-deploy/kustom/base/manifests/local/prow-rehearse-rbac.yaml`

The `--jobs-namespace` flag is set via a kustomize overlay patch in:
- `github/ci/prow-deploy/kustom/overlays/kubevirt-prow-control-plane/patches/JsonRFC6902/prow-rehearse-deployment.yaml`


## Development

Build the binary:

```bash
make build
```

Run tests:

```bash
make test
```

Format, test, and push the image:

```bash
make all
```

Push the image only:

```bash
make push
```

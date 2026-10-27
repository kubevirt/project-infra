# rehearse - external plugin for Prow

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
- Compares job configs at the PR head vs base to identify modified or new presubmit, postsubmit, and periodic jobs
- Creates rehearsal ProwJobs prefixed with `rehearsal-` for each modified or new job the commenter is allowed to rehearse

## Usage

Comment on a PR to trigger rehearsals:

```
# Rehearse all modified jobs the commenter is allowed to rehearse
/rehearse
# or
/rehearse all

# Rehearse a specific job (skipped if the commenter is not allowed to rehearse that job type)
/rehearse <job-name>

# List jobs available for rehearsal at the commenter's permission level
/rehearse ?
```

Multiple `/rehearse <job-name>` lines can be included in a single comment to rehearse several specific jobs at once.

## Authorization

Rehearsal permissions are tiered by job type:

### Presubmit jobs
A presubmit rehearsal can be triggered if the user is:
- A **top-level approver** in `project-infra`
- A **KubeVirt org member** who is a leaf approver for all files changed in the PR
- A **KubeVirt org member** if the PR has the `ok-to-rehearse` label

### Periodic and postsubmit jobs
Periodic and postsubmit rehearsals can only be triggered by **top-level approvers** in `project-infra`. No label override is available.

### Restricted jobs
Individual jobs can be excluded from rehearsal by annotating them with `rehearsal.restricted: "true"` in the job YAML. These jobs are silently skipped regardless of user permissions.

## Configuration

The plugin is registered in:
- `github/ci/prow-deploy/kustom/base/configs/current/plugins/plugins.yaml`

Deployment manifests:
- `github/ci/prow-deploy/kustom/base/manifests/local/prow-rehearse-deployment.yaml`
- `github/ci/prow-deploy/kustom/base/manifests/local/prow-rehearse-service.yaml`
- `github/ci/prow-deploy/kustom/base/manifests/local/prow-rehearse-rbac.yaml`

The `--jobs-namespace` flag is set via a kustomize overlay patch in:
- `github/ci/prow-deploy/kustom/overlays/kubevirt-prow-control-plane/patches/JsonRFC6902/prow-rehearse-deployment.yaml`

## Limitations

- Only **modified or new** jobs are eligible for rehearsal — unmodified jobs cannot be rehearsed even with `/rehearse <job-name>`
- `/rehearse` and `/rehearse all` run every changed job the commenter is allowed to rehearse. Periodics and postsubmits are included for top-level approvers and skipped (with a comment) for everyone else
- Periodic rehearsals are converted to presubmit ProwJobs so results are reported as GitHub statuses on the PR
- Rehearsals execute the original job spec, including image publishes and jobs that use secrets. Annotate a job with `rehearsal.restricted: "true"` to exclude it
- Extra-refs that match the PR repo are stripped to avoid a clone path conflict with the PR checkout
- Job configs outside the `--jobs-config-base` path are not detected and will not be rehearsed

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

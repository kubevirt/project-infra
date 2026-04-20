# phased - external plugin for Prow

## Motivation

Running expensive e2e tests on every PR update is wasteful when most PRs are
still in review. The phased plugin defers these costly tests until a PR is ready
for merging, reducing CI resource usage while still ensuring that all required
tests pass before merge.

## Overview

Phased is a Prow external plugin that automatically triggers "phase 2"
presubmit jobs when a PR on `kubevirt/kubevirt` is ready for merging. Phase 2
jobs are presubmits that require manual `/test` triggering.

When triggered, the plugin comments on the PR with `/test <job-name>` for each
phase 2 job, which Prow then picks up and runs.

## How it works

The plugin listens for `pull_request` webhook events and triggers phase 2 jobs
when specific conditions are met:

**On `labeled` events:**
- `lgtm` label added and `approved` label already exists, or
- `approved` label added and `lgtm` label already exists, or
- `skip-review` label added (regardless of other labels)

**On `synchronize` events:**
- Only if the `skip-review` label is present (`lgtm` + `approved` alone will
  not trigger phase 2 on synchronize)

**Additional conditions:**
- PR must target `main` or `master` branch
- PR must not be draft, merged, or closed
- PR must be mergeable (not in conflict)

## Phase 2 job selection

The plugin selects presubmit jobs that match all of the following criteria:
- Not optional (`optional: false`)
- Not always run (`always_run: false`)
- No `run_if_changed` condition
- No `skip_if_only_changed` condition

These are jobs that require manual `/test` triggering.

## Interaction with other components

- **Referee plugin**: The [referee](../referee/) external plugin detects phased
  plugin comments and skips them when counting test request comments, to avoid
  skewing its metrics.
- **`kubevirt-bot`**: The `skip-review` label is restricted to `kubevirt-bot`
  via Prow label restrictions in `plugins.yaml`. This is used for automated PRs
  where sufficient automated checks are in place, allowing phase 2 to trigger
  without human review labels.
- **Job configs**: The plugin reads presubmit configurations at runtime via
  HTTP from the configured Prow location (typically a raw Git URL) to identify
  phase 2 jobs.

## Configuration

The plugin is registered in:
- `github/ci/prow-deploy/kustom/base/configs/current/plugins/plugins.yaml`

Currently enabled for `kubevirt/kubevirt` only.

Deployment manifests:
- `github/ci/prow-deploy/kustom/base/manifests/local/prow-phased-deployment.yaml`
- `github/ci/prow-deploy/kustom/base/manifests/local/prow-phased-service.yaml`

## Limitations

- Hardcoded to `kubevirt/kubevirt` in the source, not just a configuration
  choice
- Only triggers for PRs targeting `main` or `master` branches

## Development

Build the binary:

```bash
make build
```

Run tests:

```bash
make test
```

Push the image:

```bash
make push
```

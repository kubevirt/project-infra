# phased - external plugin for Prow

## Motivation

Running expensive e2e tests on every PR update is wasteful when most PRs are
still in review. The phased plugin defers these costly tests until a PR is ready
for merging, reducing CI resource usage while still ensuring that all required
tests pass before merge.

The [referee](../referee/) external plugin detects phased comments and skips
them when counting test request comments, to avoid skewing its retest metrics.

## Overview

Phased is a Prow external plugin that:
- Listens for GitHub pull request webhooks (`labeled` and `synchronize` events) on `kubevirt/kubevirt`
- Triggers "phase 2" presubmit jobs when a PR targeting `main` or `master` is ready for merging
- On `labeled`: triggers when `lgtm` + `approved` are both present, or when `skip-review` is added
- On `synchronize`: triggers only when the `skip-review` label is present
- Skips PRs that are draft, merged, closed, or in a merge conflict
- Fetches Prow and presubmit job configs at runtime over HTTP from the configured `--prow-location`
- Selects presubmit jobs that are non-optional, not always-run, and have no path-based conditions (i.e. jobs requiring manual `/test` triggering)
- Posts a GitHub comment with `/test <job-name>` for each selected job, which Prow then picks up and runs

The `skip-review` label is restricted to `kubevirt-bot` and
`kubevirt-commenter-bot` via Prow label restrictions in `plugins.yaml`,
allowing automated PRs to trigger phase 2 without human review labels.

## Configuration

The plugin is registered in:
- `github/ci/prow-deploy/kustom/base/configs/current/plugins/plugins.yaml`

Currently enabled for `kubevirt/kubevirt` only.

Deployment manifests:
- `github/ci/prow-deploy/kustom/base/manifests/local/prow-phased-deployment.yaml`
- `github/ci/prow-deploy/kustom/base/manifests/local/prow-phased-service.yaml`

## Limitations

- Hardcoded to `kubevirt/kubevirt` in the source, not just a configuration choice
- Only triggers for PRs targeting `main` or `master` branches
- Does not handle `opened` events, PRs created with `skip-review` already applied will not trigger phase 2 until a subsequent `synchronize` event

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

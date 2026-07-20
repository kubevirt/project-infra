# Label Configuration Reference

For how to add a new label and the full label listing, see [labels.md](labels.md).

This document covers the configuration structure of `labels.yaml` and operational details not included in the auto-generated docs.

## Configuration structure

**Config file**: [labels.yaml](https://github.com/kubevirt/project-infra/tree/main/github/ci/prow-deploy/kustom/base/configs/current/labels/labels.yaml)

Two top-level sections:
- `default.labels` — labels applied to **all** repos in the kubevirt org
- `repos.<org>/<repo>.labels` — labels specific to a single repository

### Label fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | yes | Label name (e.g. `triage/accepted`) |
| `color` | yes | Hex color without `#` prefix (e.g. `d455d0`) |
| `target` | yes | `both` (issues + PRs), `issues`, or `prs` |
| `description` | no | **Max 100 characters** (GitHub API limit, enforced by `label_sync`) |
| `prowPlugin` | no | Prow plugin that manages this label (e.g. `label`, `lifecycle`) |
| `addedBy` | no | Who can add the label (e.g. `anyone`, `org members`, `prow`) |
| `previously` | no | List of `{name, color}` for label renames/migrations |

### Constraints

- **Label descriptions must not exceed 100 characters.** This is a GitHub API limit enforced by `label_sync`, which will reject labels that exceed it.
- Label names must be unique (case-insensitive) within each scope.

## Presubmit validation

Any PR touching `labels.yaml` triggers two dry-run presubmit jobs that validate the config without applying it:
- `pull-prow-kubevirt-labels-update-precheck` (kubevirt org)
- `pull-prow-nmstate-labels-update-precheck` (nmstate repo)

## Dependency ordering

When a change depends on a new label existing on GitHub (e.g. a job config that applies the label), the label definition PR must merge first **and** the daily CronJob must run before the dependent change will work. Use `/hold` on the dependent PR until the label is available.

# retester

Posts `/retest-required` (or a skip comment) on merge-ready PRs whose **required** presubmits are failing.

Must be run from a **project-infra** checkout: it loads `github/ci/prow-deploy/files/jobs/kubevirt/{kubevirt,kubevirtci,project-infra}/*-presubmits*.yaml`.

## Usage

```bash
retester [flags]
```

## Flags

- `--updated` — only PRs whose `updated` time is at least this old (default: `2h`; added to the search as `updated:<=RFC3339`)
- `--confirm` — mutate GitHub; if false, uses Prow’s dry-run GitHub client (default: `false`)
- `--ceiling` — max issues to comment on **per search query** (`0` = unlimited; default: `1`)
- `--token` — path to GitHub token (**required**; empty fatals)
- `--endpoint` — GitHub REST API URL; repeatable (default: `https://api.github.com`)
- `--graphql-endpoint` — GitHub GraphQL endpoint (default: `https://api.github.com/graphql`)

## How it works

Two searches (each combined with the repo list `repo:kubevirt/kubevirt`, `repo:kubevirt/kubevirtci`, `repo:kubevirt/project-infra`):

- `label:lgtm label:approved`
- `label:skip-review`

Plus: open, unlocked, not archived, `status:failure`, and exclusions for `needs-rebase`, bare `do-not-merge`, and several `do-not-merge/*` labels (hold, work-in-progress, invalid-owners-file, etc.).

For each hit, up to `--ceiling` per query:

1. Combined status on the PR head SHA.
2. Failures whose Prow URL job name is in the required-presubmit map.
3. If none of those are required failures, skip the PR (no comment).
4. If `notFitForRetest` returns a reason, post the skip template (skipped if the PR’s **last** comment is already that skip body).
5. Otherwise post the `/retest-required` comment. **Retest comments are not de-duplicated.**

Because there are two queries, `--ceiling=1` can still comment on two PRs per process (one per query).

## Required presubmits

A job is required if `AlwaysRun || RunBeforeMerge`, `Optional` is false, and both `RunIfChanged` and `SkipIfOnlyChanged` are empty.

## Skip retest when

**Deterministic job names** (regex `-build(-|$)`, `-generate$`, `-check-tests-for-flakes$`): retest will not help.

**E2E SIG lane failed on every k8s version seen** (at least two versions among required `*-e2e-k8s-<ver>-<lane>` statuses that are success or failure). Flakes usually hit one version, not all.

## Comment bodies

Retest:

```
/retest-required
This bot automatically retries required jobs that failed/flaked on 
required test lanes of PRs.
Silence the bot with an `/lgtm cancel` or `/hold` comment for consistent failures.
```

Skip inserts the reason string into:

```
Retesting this PR is being skipped.

<reason>

This bot automatically retries jobs that failed on required test lanes, but
skips PRs where failures indicate issues that a retest would not fix.
Silence the bot with an `/lgtm cancel` or `/hold` comment.
```

## Integration

`periodic-project-infra-retester` runs **hourly** (`interval: 1h`) on `kubevirt-prow-control-plane`:

```bash
go run ./robots/retester --token=/etc/github/token --ceiling=1 --confirm
```

Token secret: `commenter-oauth-token` mounted at `/etc/github/token`.

## Example

```bash
./retester --token=/path/to/token --confirm=false --ceiling=1 --updated=2h
```

## Development

```bash
cd robots/retester
go build -o retester .
go test -v
```

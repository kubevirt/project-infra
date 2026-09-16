# labels-checker

Fails if an **open** PR matching org/repo/author/branch has any of a given set of labels.

Used from `hack/git-pr.sh` when `--missing-labels` is set (built into `quay.io/kubevirtci/pr-creator` as `/usr/local/bin/labels-checker`) so automation can skip updating a bot PR that already has review labels such as `lgtm` or `approved`.

## Usage

```bash
labels-checker [flags]
```

## Flags

- `--org` — GitHub org (required)
- `--repo` — GitHub repo (required)
- `--author` — PR head owner (required); used as GitHub `Head` `author:branch-name`
- `--branch-name` — PR head branch (required)
- `--ensure-labels-missing` — comma-separated labels that must **not** be on the PR (default: `lgtm`)
- `--github-token-path` — token file (default: `/etc/github/token`). Empty path = unauthenticated client.
- `--github-endpoint` — GitHub API endpoint (default: `https://api.github.com/`)

## How it works

1. `PullRequests.List` with `state=open` and `head=author:branch-name`.
2. **Zero PRs:** logs `No PR found` and **exits 0** (success). `git-pr.sh` then continues and can open a new PR.
3. **More than one PR:** fatals.
4. **One PR:** fatals if any `--ensure-labels-missing` label is present; otherwise exits 0.

## Exit codes

- `0` — no matching PR, or the PR exists and none of the forbidden labels are present
- non-zero — invalid flags, API error, multiple PRs, or a forbidden label is present

## Example (same pattern as `hack/git-pr.sh`)

```bash
labels-checker \
  --org=kubevirt \
  --repo=project-infra \
  --author=kubevirt-bot \
  --branch-name=project-infra-image-bump \
  --ensure-labels-missing=lgtm,approved,do-not-merge/hold,skip-review \
  --github-token-path=/etc/github/token
```

## Development

```bash
cd robots/labels-checker
go build -o labels-checker .
go test -v
```

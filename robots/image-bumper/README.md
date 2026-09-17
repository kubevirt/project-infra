# image-bumper

Updates kubevirtci and related container image references in this repository. Logic lives in `pkg/imagebump/`.

The cobra root `Use` (help text) is `bump`. Subcommands hang off the root; invoke them on the compiled binary or via `go run`. Do **not** insert an extra `bump` token (`image-bumper bump all` is invalid):

```bash
go run ./robots/image-bumper <subcommand> [--repo-root <path>]
```

## Commands

| Command | What it updates |
| --- | --- |
| `job-images` | `quay.io/kubevirtci/*` tags in `*-periodics.yaml` / `*-presubmits.yaml` / `*-postsubmits.yaml` under `github/ci/prow-deploy/files/jobs` (optional `-master`/`-main` suffix; not versioned `*-presubmits-1.6.yaml`; skips `bootstrap-legacy`) |
| `prow-deployment-images` | `quay.io/kubevirtci/*` tags in `github/ci/prow-deploy/kustom/base/manifests/local` and `github/ci/prow-deploy/kustom/overlays/prow-workloads/resources` |
| `containerfile-images` | `FROM` lines in **git-tracked** `Containerfile`/`Dockerfile` files that use `quay.io/...:tag` |
| `all` | The three operations above, resolving kubevirtci tags once and reusing that map for job and deployment bumps |

## Flags

- `--repo-root` — path to the project-infra git checkout (default: `.`)

## Requirements

- `job-images`, `prow-deployment-images`, and `all` require `skopeo` on `PATH` (list-tags / inspect).
- `containerfile-images` uses the Quay HTTP API (`/tag/?limit=1`), not skopeo.

## How it works

1. **Job and deployment bumps:** scan `github/ci/prow-deploy` YAML for `quay.io/kubevirtci/<repo>` names, ask skopeo for the latest tag matching `^v?[0-9]+-[a-z0-9]{7,9}$`, then do in-place string replace of `:tag` or `@sha256:…` (YAML formatting is preserved).
2. **Containerfile bumps:** `git ls-files` for Containerfile/Dockerfile paths, then replace `FROM` image tags via Quay. Non-quay images, digest pins, and tags Quay does not return are skipped.

## Integration

`periodic-project-infra-image-bump` (daily `30 0 * * *`) runs:

```bash
go run ./robots/image-bumper all --repo-root=$(pwd)
```

inside `hack/git-pr.sh`, which opens a project-infra PR when files change.

Related shell helpers (manual, per-image): `hack/update-jobs-with-latest-image.sh`, `hack/update-deployments-with-latest-image.sh`, `hack/update-containerfiles-with-latest-baseimage.sh`.

## Example

```bash
cd /path/to/project-infra
go run ./robots/image-bumper all
git diff
```

## Development

```bash
cd robots/image-bumper
go build -o image-bumper .
make test   # go test of this package and pkg/imagebump
```

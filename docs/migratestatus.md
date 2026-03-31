# migratestatus

`migratestatus` is a maintenance tool (borrowed from `kubernetes/test-infra`) used to safely migrate or retire GitHub
commit status contexts on open PRs. It is packaged as a container image and wrapped by the `hack/migratestatus.sh`
script.

## When to use

Use this tool during [lane migrations](lane-migration-process.md) to retire old status contexts after a CI job has been
renamed or removed. This prevents stale required statuses from blocking PRs.

## Usage

```
hack/migratestatus.sh <gh-token-path> [flags...]
```

The script mounts the token file into the container and passes all remaining arguments to the `migratestatus` binary.

### Modes

Exactly one mode must be specified per invocation.

#### Retire

Set an old context's state to "success" so it no longer blocks PRs.

- **With replacement**: marks the old context as retired and points to the replacement.
- **Without replacement**: marks the old context as retired without a replacement.

```bash
# Retire without replacement (dry-run)
hack/migratestatus.sh /path/to/oauth \
  --retire 'pull-kubevirt-e2e-k8s-1.31-sig-compute-migrations' \
  --org kubevirt --repo kubevirt \
  --branch-filter main \
  --dry-run=true

# Retire with replacement
hack/migratestatus.sh /path/to/oauth \
  --retire 'pull-kubevirt-e2e-k8s-1.31-sig-compute-migrations' \
  --dest 'pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations' \
  --org kubevirt --repo kubevirt \
  --branch-filter main \
  --dry-run=false
```

#### Copy

Copy an existing context's status to a new destination context. Useful for setting up a new context before retiring the
old one.

```bash
hack/migratestatus.sh /path/to/oauth \
  --copy 'pull-kubevirt-e2e-k8s-1.31-sig-compute-migrations' \
  --dest 'pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations' \
  --org kubevirt --repo kubevirt \
  --branch-filter main \
  --dry-run=true
```

#### Move

Copy and retire in a single step. The old context's status is copied to the destination, then the old context is retired
with the destination as its replacement.

```bash
hack/migratestatus.sh /path/to/oauth \
  --move 'pull-kubevirt-e2e-k8s-1.31-sig-compute-migrations' \
  --dest 'pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations' \
  --org kubevirt --repo kubevirt \
  --branch-filter main \
  --dry-run=true
```

### Flags

| Flag                  | Description                                                       | Default |
|-----------------------|-------------------------------------------------------------------|---------|
| `--copy`              | Copy mode: context to copy                                        |         |
| `--move`              | Move mode: context to move                                        |         |
| `--retire`            | Retire mode: context to retire                                    |         |
| `--dest`              | Destination context (required for copy/move, optional for retire) |         |
| `--org`               | GitHub organization                                               |         |
| `--repo`              | GitHub repository                                                 |         |
| `--branch-filter`     | Regex to match target branch (optional)                           |         |
| `--description`       | URL explaining the migration (optional, not for copy mode)        |         |
| `--continue-on-error` | Continue if migration fails for individual PRs                    | `false` |
| `--dry-run`           | Preview changes without modifying anything                        | `true`  |

### Dry-run

The tool runs in **dry-run mode by default**. Always verify the output with `--dry-run=true` before running with
`--dry-run=false`.

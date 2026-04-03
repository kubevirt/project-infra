# periodic-jobs

A CLI tool for managing Prow periodic job schedules with two subcommands:

- **`gantt`** - Generate Mermaid Gantt charts to visualize periodic job schedules
- **`spread`** - Spread periodic jobs evenly across time slots to reduce load clustering

## Shared Flags

Both subcommands share these flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--input` | `github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml` | Input YAML file path |
| `--pattern` | `periodic-kubevirt-e2e-k8s-` | Job name prefix to match |

## Gantt Subcommand

Generates a Mermaid Gantt chart showing periodic job schedules over a 24-hour period, grouped by SIG.

### Usage

```bash
go run ./cmd/periodic-jobs gantt [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--runtimes` | (embedded defaults) | Custom runtimes YAML file |

### Examples

```bash
# Generate chart with defaults
go run ./cmd/periodic-jobs gantt

# Use a custom input file
go run ./cmd/periodic-jobs gantt --input path/to/periodics.yaml

# Use custom runtime estimates
go run ./cmd/periodic-jobs gantt --runtimes my-runtimes.yaml
```

### Output

Outputs a Mermaid code block ready to paste into GitHub comments, PRs, or markdown files.

### Runtime Estimates

The tool includes embedded default runtime estimates (`default-runtimes.yaml`). Override with `--runtimes`:

```yaml
runtimes:
  sig-compute: 4.0
  sig-network: 2.5
default: 2.0
```

## Spread Subcommand

Redistributes periodic job cron schedules to minimize concurrent execution.

### Algorithm

1. Groups jobs by frequency (times per day)
2. Calculates the period between runs (e.g., 4x/day = 6h period)
3. Staggers jobs evenly within each period
4. Updates cron expressions with new times

### Usage

```bash
go run ./cmd/periodic-jobs spread [flags]
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--output` | (same as input) | Output YAML file path |
| `--dry-run` | `false` | Print changes without modifying files |
| `--verbose` | `false` | Enable verbose output |

### Examples

```bash
# Dry run to preview changes
go run ./cmd/periodic-jobs spread --dry-run --verbose

# Spread jobs and write back
go run ./cmd/periodic-jobs spread --verbose

# Write to a different file
go run ./cmd/periodic-jobs spread --output spread-periodics.yaml
```

## Makefile Targets

```bash
make periodic-jobs-gantt          # Generate Gantt chart
make periodic-jobs-spread         # Spread jobs (modifies file)
make periodic-jobs-spread-dry-run # Preview spread changes
```

## Workflow

```bash
# 1. Visualize current schedule
make periodic-jobs-gantt > before.md

# 2. Spread the jobs
make periodic-jobs-spread

# 3. Visualize new schedule
make periodic-jobs-gantt > after.md
```

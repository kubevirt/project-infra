# Spread Periodic Jobs

A tool to automatically spread periodic Prow job schedules evenly across the day to reduce load clustering.

## Overview

This tool analyzes periodic jobs in a Prow configuration file and redistributes their cron schedules to minimize concurrent job execution. It groups jobs by frequency (how many times per day they run) and applies an optimal spreading algorithm.

## Algorithm

The spreading strategy works as follows:

1. **Group by Frequency**: Jobs are grouped by how many times they run per day (2x, 3x, 4x, etc.)

2. **Calculate Period**: For each frequency group, determine the period between runs:
   - 4×/day → 6 hour period
   - 3×/day → 8 hour period
   - 2×/day → 12 hour period

3. **Calculate Stagger**: Divide one period by the number of jobs to get the stagger interval:
   - Example: 14 jobs at 4×/day → 6 hours ÷ 14 jobs = ~25 minutes between starts

4. **Assign Times**: Assign each job a start time offset within the period:
   - Job 1: 0:00, 6:00, 12:00, 18:00
   - Job 2: 0:25, 6:25, 12:25, 18:25
   - Job 3: 0:50, 6:50, 12:50, 18:50
   - etc.

This ensures jobs are evenly distributed throughout the day, avoiding clustering at specific times.

## Usage

### Basic Usage

```bash
go run ./cmd/spread-periodic-jobs \
  --input github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml
```

### Options

- `--input <file>`: Input YAML file path (required)
- `--output <file>`: Output YAML file path (defaults to input file)
- `--pattern <pattern>`: Job name pattern to match (default: `periodic-kubevirt-e2e-k8s-`)
- `--dry-run`: Print changes without modifying files
- `--verbose`: Enable verbose output showing all job assignments

### Examples

```bash
# Dry run to see what would change
go run ./cmd/spread-periodic-jobs \
  --input kubevirt-periodics.yaml \
  --dry-run \
  --verbose

# Spread only specific jobs
go run ./cmd/spread-periodic-jobs \
  --input kubevirt-periodics.yaml \
  --pattern "periodic-kubevirt-e2e-k8s-1.3"

# Write to a different file
go run ./cmd/spread-periodic-jobs \
  --input kubevirt-periodics.yaml \
  --output kubevirt-periodics-spread.yaml
```

## Example Output

```
Found 21 jobs matching pattern 'periodic-kubevirt-e2e-k8s-'

Spreading 14 jobs at 4x/day (every 6h):
  Stagger interval: 25 minutes

Spreading 3 jobs at 3x/day (every 8h):
  Stagger interval: 160 minutes

Spreading 4 jobs at 2x/day (every 12h):
  Stagger interval: 180 minutes

Successfully updated kubevirt-periodics.yaml

Cron expression changes:
  periodic-kubevirt-e2e-k8s-1.31-sig-network: 0 0,6,12,18 * * *
  periodic-kubevirt-e2e-k8s-1.31-sig-operator: 25 0,6,12,18 * * *
  periodic-kubevirt-e2e-k8s-1.31-sig-storage: 50 0,6,12,18 * * *
  ...
```

## How It Works

The tool:

1. Parses the YAML file while preserving formatting and comments
2. Finds all jobs matching the specified pattern
3. Groups jobs by their execution frequency (number of times per day)
4. Calculates optimal stagger intervals for each group
5. Updates cron expressions with new evenly-distributed times
6. Writes the modified YAML back to disk

## Implementation Details

- Uses `gopkg.in/yaml.v3` to preserve YAML formatting and comments
- Parses cron expressions to determine job frequency
- Handles comma-separated hour lists (e.g., `1,7,13,19`)
- Maintains alphabetical sorting within frequency groups for deterministic output
- Preserves all other job configuration unchanged

## Benefits

- **Reduced Peak Load**: Eliminates clustering of job starts
- **Predictable Load**: Spreads jobs evenly throughout the day
- **Resource Efficiency**: Better utilization of compute resources
- **Automation**: No manual calculation of cron schedules needed

## Real-World Results

In the kubevirt/kubevirt project, this tool reduced:
- Peak concurrent jobs from 17 → 11
- Number of high-load windows (≥12 concurrent jobs) from 9 → 0
- Load variance: all 48 thirty-minute windows now stay within 8–11 concurrent jobs

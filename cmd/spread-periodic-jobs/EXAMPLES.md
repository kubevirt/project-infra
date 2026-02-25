# Practical Examples

This document provides practical examples of using the periodic job spreading tool.

## Example 1: Basic Usage

### Scenario

You have a Prow configuration file with 20 periodic jobs that are clustered around similar times, causing resource contention.

### Steps

1. **Backup your file** (always a good practice):

```bash
cp kubevirt-periodics.yaml kubevirt-periodics.yaml.backup
```

2. **Run in dry-run mode** to see what would change:

```bash
go run ./cmd/spread-periodic-jobs \
  --input github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml \
  --dry-run \
  --verbose
```

3. **Review the output** to ensure the changes look reasonable:

```
Found 23 jobs matching pattern 'periodic-kubevirt-e2e-k8s-'

Spreading 14 jobs at 4x/day (every 6h):
  Stagger interval: 25 minutes
  periodic-kubevirt-e2e-k8s-1.33-sig-compute: 0 0,6,12,18 * * *
  periodic-kubevirt-e2e-k8s-1.33-sig-network: 25 0,6,12,18 * * *
  ...

Spreading 3 jobs at 3x/day (every 8h):
  Stagger interval: 160 minutes
  ...
```

4. **Apply the changes**:

```bash
go run ./cmd/spread-periodic-jobs \
  --input github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml
```

5. **Verify the changes**:

```bash
git diff github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml
```

## Example 2: Targeting Specific Jobs

### Scenario

You only want to spread e2e jobs for Kubernetes 1.35, not all e2e jobs.

### Solution

Use a more specific pattern:

```bash
go run ./cmd/spread-periodic-jobs \
  --input kubevirt-periodics.yaml \
  --pattern "periodic-kubevirt-e2e-k8s-1.35" \
  --dry-run
```

This will only match jobs like:
- `periodic-kubevirt-e2e-k8s-1.35-sig-compute`
- `periodic-kubevirt-e2e-k8s-1.35-sig-network`
- etc.

## Example 3: Multiple Pattern Runs

### Scenario

You want to spread different categories of jobs with different strategies.

### Solution

Run the tool multiple times with different patterns:

```bash
# Spread 1.33 jobs
go run ./cmd/spread-periodic-jobs \
  --input kubevirt-periodics.yaml \
  --pattern "periodic-kubevirt-e2e-k8s-1.33"

# Spread 1.34 jobs
go run ./cmd/spread-periodic-jobs \
  --input kubevirt-periodics.yaml \
  --pattern "periodic-kubevirt-e2e-k8s-1.34"

# Spread 1.35 jobs
go run ./cmd/spread-periodic-jobs \
  --input kubevirt-periodics.yaml \
  --pattern "periodic-kubevirt-e2e-k8s-1.35"
```

**Note**: Each run modifies the file, so patterns should be mutually exclusive to avoid conflicts.

## Example 4: Creating a Testing Environment

### Scenario

You want to test the spread schedule before applying to production.

### Solution

1. **Create a test copy**:

```bash
cp kubevirt-periodics.yaml kubevirt-periodics-test.yaml
```

2. **Apply spreading to test file**:

```bash
go run ./cmd/spread-periodic-jobs \
  --input kubevirt-periodics-test.yaml \
  --output kubevirt-periodics-spread.yaml
```

3. **Compare the results**:

```bash
# Show only cron changes
diff <(grep -E "name:|cron:" kubevirt-periodics.yaml) \
     <(grep -E "name:|cron:" kubevirt-periodics-spread.yaml)
```

4. **Analyze load distribution** (see Analysis Scripts below)

## Example 5: Integration with CI/CD

### Scenario

You want to automatically spread jobs as part of a CI pipeline.

### GitHub Actions Example

```yaml
name: Spread Periodic Jobs

on:
  pull_request:
    paths:
      - 'github/ci/prow-deploy/files/jobs/**/periodics.yaml'

jobs:
  spread-jobs:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Setup Go
        uses: actions/setup-go@v2
        with:
          go-version: '1.25'

      - name: Spread periodic jobs
        run: |
          go run ./cmd/spread-periodic-jobs \
            --input github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml \
            --dry-run \
            --verbose

      - name: Comment on PR
        uses: actions/github-script@v5
        with:
          script: |
            github.rest.issues.createComment({
              issue_number: context.issue.number,
              owner: context.repo.owner,
              repo: context.repo.repo,
              body: 'Periodic jobs have been analyzed. Review the output above.'
            })
```

## Example 6: Makefile Integration

### Scenario

You want a simple command to spread jobs.

### Makefile

```makefile
.PHONY: spread-periodic-jobs
spread-periodic-jobs:
	@echo "Spreading periodic jobs..."
	go run ./cmd/spread-periodic-jobs \
		--input github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml \
		--verbose

.PHONY: spread-periodic-jobs-dry-run
spread-periodic-jobs-dry-run:
	@echo "Dry run: spreading periodic jobs..."
	go run ./cmd/spread-periodic-jobs \
		--input github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml \
		--dry-run \
		--verbose

.PHONY: analyze-periodic-load
analyze-periodic-load:
	@echo "Analyzing periodic job load distribution..."
	@./scripts/analyze-periodic-load.sh
```

Usage:

```bash
make spread-periodic-jobs-dry-run
make spread-periodic-jobs
make analyze-periodic-load
```

## Analysis Scripts

### Script 1: Count Jobs Per Hour

Create `scripts/analyze-periodic-load.sh`:

```bash
#!/bin/bash
# Count how many jobs start in each hour of the day

PERIODICS_FILE="${1:-github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml}"

echo "Job start times per hour:"
echo "Hour | Count | Jobs"
echo "-----|-------|-----"

for hour in {0..23}; do
  jobs=$(grep -E "cron:.*( |,)${hour}(,| )" "$PERIODICS_FILE" | wc -l)
  bar=$(printf '=%.0s' $(seq 1 $jobs))
  printf "%2d   | %5d | %s\n" $hour $jobs "$bar"
done
```

Usage:

```bash
chmod +x scripts/analyze-periodic-load.sh
./scripts/analyze-periodic-load.sh
```

Expected output:

```
Job start times per hour:
Hour | Count | Jobs
-----|-------|-----
 0   |     4 | ====
 1   |     3 | ===
 2   |     4 | ====
 3   |     4 | ====
 4   |     3 | ===
 5   |     3 | ===
 6   |     4 | ====
 7   |     3 | ===
...
```

### Script 2: Visualize Concurrent Jobs

Create `scripts/visualize-concurrency.sh`:

```bash
#!/bin/bash
# Visualize concurrent jobs across the day
# This creates a simple ASCII visualization

PERIODICS_FILE="${1:-github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml}"

echo "Concurrent jobs visualization (assuming 30-min duration):"
echo "Time  | Concurrent"
echo "------|--------------------"

for hour in {0..23}; do
  for minute in 0 30; do
    # Count jobs starting in this 30-minute window
    count=0

    # This is a simplified version - real script would parse cron expressions
    if [ $minute -eq 0 ]; then
      count=$(grep -E "cron:.*( |,)${hour}(,| )" "$PERIODICS_FILE" | wc -l)
    fi

    printf "%02d:%02d | " $hour $minute
    printf '#%.0s' $(seq 1 $count)
    printf " (%d)\n" $count
  done
done
```

### Script 3: Load Distribution Stats

Create `scripts/load-stats.sh`:

```bash
#!/bin/bash
# Calculate load distribution statistics

PERIODICS_FILE="${1:-github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml}"

declare -A hour_counts

# Count jobs per hour
for hour in {0..23}; do
  count=$(grep -E "cron:.*( |,)${hour}(,| )" "$PERIODICS_FILE" | wc -l)
  hour_counts[$hour]=$count
done

# Calculate statistics
min=${hour_counts[0]}
max=${hour_counts[0]}
sum=0

for hour in {0..23}; do
  count=${hour_counts[$hour]}
  sum=$((sum + count))

  if [ $count -lt $min ]; then
    min=$count
  fi

  if [ $count -gt $max ]; then
    max=$count
  fi
done

avg=$(echo "scale=2; $sum / 24" | bc)
variance=0

echo "Load Distribution Statistics:"
echo "  Minimum jobs per hour: $min"
echo "  Maximum jobs per hour: $max"
echo "  Average jobs per hour: $avg"
echo "  Total job starts per day: $sum"
echo "  Load variance: $((max - min))"

if [ $max -gt 0 ]; then
  flatness=$(echo "scale=2; $max / $avg" | bc)
  echo "  Flatness ratio: $flatness (ideal: 1.0)"
fi
```

## Advanced Example: Pre-commit Hook

### Scenario

You want to ensure periodic jobs are always spread when changes are committed.

### Setup

1. Create `.git/hooks/pre-commit`:

```bash
#!/bin/bash

# Check if kubevirt-periodics.yaml was modified
if git diff --cached --name-only | grep -q "kubevirt-periodics.yaml"; then
  echo "Detected changes to kubevirt-periodics.yaml"
  echo "Running periodic job spreading..."

  # Run spreading
  go run ./cmd/spread-periodic-jobs \
    --input github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml \
    --verbose

  # Stage the changes
  git add github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml

  echo "Periodic jobs have been spread automatically"
fi
```

2. Make it executable:

```bash
chmod +x .git/hooks/pre-commit
```

**Note**: This is aggressive and may not be desired in all workflows. Consider using a separate command instead.

## Troubleshooting

### Problem: Jobs aren't being spread evenly

**Diagnosis**: Check if jobs have different frequencies

```bash
# Extract all cron expressions
grep -E "name: periodic-kubevirt-e2e|cron:" kubevirt-periodics.yaml | \
  grep -A1 "periodic-kubevirt-e2e" | \
  grep cron | \
  awk -F': ' '{print $2}' | \
  sort | uniq -c
```

**Solution**: Jobs with different frequencies are spread separately. This is by design.

### Problem: Tool modifies non-matching jobs

**Diagnosis**: Check your pattern

```bash
# List all jobs that match your pattern
grep "name:.*periodic-kubevirt-e2e-k8s-" kubevirt-periodics.yaml
```

**Solution**: Use a more specific pattern or adjust job names.

### Problem: YAML formatting is changed

**Solution**: The tool uses `gopkg.in/yaml.v3` which preserves most formatting, but some minor differences may occur. Use `git diff` to review changes.

## Best Practices

1. **Always backup** before making changes
2. **Use dry-run first** to review changes
3. **Test in a separate branch** before applying to main
4. **Run analysis scripts** to verify improvements
5. **Document your pattern choice** in commit messages
6. **Consider frequency groups** when analyzing results
7. **Monitor actual load** after deployment to validate improvements

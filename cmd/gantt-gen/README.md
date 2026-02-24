# gantt-gen

A tool to visualize periodic job schedules as Mermaid Gantt charts.

## Overview

`gantt-gen` is a flexible CLI tool that reads Prow periodic job configurations and generates Mermaid Gantt charts to visualize job schedules over a 24-hour period. Jobs are grouped by SIG and annotated with estimated runtimes.

By default, it reads `kubevirt-periodics.yaml` and visualizes `periodic-kubevirt-e2e-k8s-*` jobs, but it can be configured to work with any project's periodic jobs using command-line flags.

This visualization helps identify:
- Load distribution across the day
- Overlapping job execution windows
- Peak resource usage times
- Gaps in the schedule

## Usage

### Basic Usage

Run from the repository root:

```bash
go run ./cmd/gantt-gen/
```

This reads the default path: `github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml`

### Command-Line Options

```bash
go run ./cmd/gantt-gen/ [options]

Options:
  --input <file>      Input YAML file path (default: kubevirt-periodics.yaml)
  --pattern <prefix>  Job name prefix to match (default: "periodic-kubevirt-e2e-k8s-")
  --runtimes <file>   Custom runtimes YAML file (optional, uses embedded defaults)
```

### Custom YAML File

```bash
go run ./cmd/gantt-gen/ --input path/to/your/periodics.yaml
```

### Custom Job Pattern

```bash
# Visualize only specific jobs
go run ./cmd/gantt-gen/ --pattern "periodic-my-project-"

# Combine with custom file
go run ./cmd/gantt-gen/ \
  --input path/to/periodics.yaml \
  --pattern "periodic-custom-"
```

### Custom Runtime Estimates

The tool uses embedded default runtime estimates (see `default-runtimes.yaml`). You can override these with your own estimates:

```bash
# Create your custom runtimes file
cat > my-runtimes.yaml <<EOF
runtimes:
  sig-compute: 5.0
  sig-network: 3.0
  sig-storage: 4.5
  sig-operator: 2.5
default: 2.0
EOF

# Use custom runtimes
go run ./cmd/gantt-gen/ --runtimes my-runtimes.yaml
```

The runtimes file format:
```yaml
# Runtime estimates in hours
runtimes:
  sig-name-1: 3.5
  sig-name-2: 2.0
  # Add more as needed

# Fallback for unmatched SIGs
default: 2.0
```

### Output

The tool outputs a Mermaid code block ready to paste into:
- GitHub comments
- Pull request descriptions
- Issue descriptions
- Markdown documentation

Example output:

````markdown
```mermaid
gantt
    title periodic-kubevirt-e2e-k8s-* Schedule (24h)
    dateFormat HH:mm
    axisFormat %H:%M
    tickInterval 3h

    section sig-compute
    1.33       :00:00, 4h
    1.34       :02:55, 4h
    1.35       :04:35, 4h

    section sig-network
    1.33       :00:25, 2h30m
    1.35       :03:45, 2h30m
    ...
```
````

## How It Works

1. **Load Configuration**: Reads custom or embedded default runtime estimates
2. **Parse YAML**: Reads the Prow periodic jobs configuration file
3. **Filter Jobs**: Selects jobs matching the specified pattern (default: `periodic-kubevirt-e2e-k8s-*`)
4. **Extract Info**: Parses job names to extract version and SIG information
5. **Parse Cron**: Extracts start times from cron expressions
6. **Estimate Runtime**: Assigns runtime duration using the loaded runtimes configuration
7. **Group & Sort**: Groups by SIG, sorts by start time within each group
8. **Generate Gantt**: Creates Mermaid Gantt chart syntax with all job runs

## Runtime Estimates

The tool uses these **default** estimated runtimes per SIG (defined in `default-runtimes.yaml`):

| SIG | Estimated Runtime |
|-----|-------------------|
| sig-compute-migrations | 3.0 hours |
| sig-compute-root | 4.0 hours |
| sig-compute | 4.0 hours |
| sig-operator-root | 2.0 hours |
| sig-operator | 2.0 hours |
| sig-storage-root | 3.5 hours |
| sig-storage | 3.5 hours |
| sig-network-with-dnc | 2.5 hours |
| sig-network | 2.5 hours |
| sig-monitoring | 1.5 hours |
| sig-performance | 2.0 hours |
| **Default fallback** | 2.0 hours |

**Note**: These are estimates. Actual runtimes may vary. You can provide custom runtime estimates using the `--runtimes` flag (see [Custom Runtime Estimates](#custom-runtime-estimates)).

## Job Name Format

By default, the tool expects job names starting with `periodic-kubevirt-e2e-k8s-` followed by:

```
<pattern><version>-<sig-name>
```

Examples:
- `periodic-kubevirt-e2e-k8s-1.35-sig-compute`
- `periodic-kubevirt-e2e-k8s-1.34-sig-network`
- `periodic-kubevirt-e2e-k8s-1.33-ipv6-sig-network`

The tool splits on `-sig-` to extract:
- **Version**: Everything before `-sig-` (e.g., `1.35`, `1.34-ipv6`)
- **SIG**: Everything from `-sig-` onwards (e.g., `sig-compute`, `sig-network`)

**Custom Patterns**: Use `--pattern` to visualize jobs with different naming conventions.

## Example Workflow

### Visualize Current Schedule

```bash
# Generate Gantt chart
go run ./cmd/gantt-gen/ > schedule.md

# View in GitHub
# Copy the output and paste into a GitHub comment/PR
```

### Compare Before/After Spreading

```bash
# Before spreading
go run ./cmd/gantt-gen/ > before.md

# Spread the jobs
make spread-periodic-jobs

# After spreading
go run ./cmd/gantt-gen/ > after.md

# Compare the visualizations
diff before.md after.md
```

### Visualize Different Projects

```bash
# Visualize a different project's jobs with custom runtimes
go run ./cmd/gantt-gen/ \
  --input github/ci/prow-deploy/files/jobs/other-org/other-repo/periodics.yaml \
  --pattern "periodic-other-project-" \
  --runtimes other-project-runtimes.yaml

# Or with defaults (if job naming follows -sig- convention)
go run ./cmd/gantt-gen/ \
  --input github/ci/prow-deploy/files/jobs/other-org/other-repo/periodics.yaml \
  --pattern "periodic-other-project-"
```

### Use in Pull Requests

When creating a PR that changes job schedules:

1. Generate the Gantt chart
2. Include it in the PR description
3. Reviewers can visualize the changes

Example PR description:

````markdown
## Job Schedule Visualization

```mermaid
gantt
    title periodic-kubevirt-e2e-k8s-* Schedule (24h)
    ...
```

This shows the new distribution after spreading jobs evenly.
````

## Integration

### Makefile Target

Add to your Makefile:

```makefile
.PHONY: visualize-periodic-jobs
visualize-periodic-jobs:
	@go run ./cmd/gantt-gen/
```

Usage:
```bash
make visualize-periodic-jobs > schedule.md
```

### CI/CD

Generate visualizations automatically in CI:

```yaml
- name: Visualize job schedule
  run: |
    go run ./cmd/gantt-gen/ > /tmp/schedule.md
    gh pr comment ${{ github.event.pull_request.number }} \
      --body-file /tmp/schedule.md
```

## Files in This Directory

- **`main.go`** (229 lines): Main program with CLI interface and Gantt chart generation logic
- **`default-runtimes.yaml`** (20 lines): Embedded default runtime estimates for kubevirt SIGs
- **`README.md`** (this file): Comprehensive usage documentation

### About the Embedded Configuration

The `default-runtimes.yaml` file is embedded in the binary using Go's `go:embed` directive:
- No external file dependencies at runtime
- Tool works as a standalone binary
- Default configuration is always available
- You can view/copy this file to create custom configurations
- Custom runtimes via `--runtimes` override embedded defaults

## Limitations

- Job pattern matching: By default targets `periodic-kubevirt-e2e-k8s-*` jobs (fully configurable via `--pattern`)
- Runtime estimates: Approximate values used for visualization (customize with `--runtimes` for your project)
- Queue time: Does not account for queue wait times before job execution
- Concurrency: Shows scheduled start times, not actual concurrent execution windows
- Time period: Displays a single 24-hour period (periodic jobs repeat on subsequent days)
- SIG extraction: Expects job names to contain `-sig-` separator (works best with standardized naming)

## Mermaid Rendering

Mermaid Gantt charts can be rendered by:
- **GitHub**: Automatically in comments, PRs, issues, markdown files
- **GitLab**: In markdown files
- **VS Code**: With Mermaid extension
- **Online**: https://mermaid.live/

## Dependencies

Uses the `github.com/nao1215/markdown` library for Mermaid generation:
- Clean Go API for Gantt charts
- No external rendering dependencies
- Outputs standard Mermaid syntax

## Tips & Troubleshooting

### Viewing the Embedded Defaults

To see the embedded default runtimes configuration:

```bash
# The file is in the repository
cat cmd/gantt-gen/default-runtimes.yaml
```

### Creating Runtime Estimates

To determine appropriate runtime estimates for your project:

1. Check actual job durations in your CI dashboard
2. Use median or 75th percentile times (not max, to avoid outliers)
3. Round to convenient intervals (0.5h, 1h, 1.5h, etc.)
4. Group similar job types with the same estimate

### Pattern Matching

The `--pattern` flag does simple prefix matching. For jobs like:
- `periodic-myproject-e2e-test-1.30` → use `--pattern "periodic-myproject-"`
- `nightly-integration-suite-v2` → use `--pattern "nightly-integration-"`

### SIG Extraction

The tool splits on `-sig-` to extract SIG names. If your jobs don't follow this convention:
- Jobs will be grouped under "other"
- Consider renaming jobs or adjusting the pattern to group jobs differently

### Large Outputs

For projects with many jobs (>50), consider:
- Filtering by pattern to visualize subsets
- Using a wider terminal or saving to file
- Adjusting Mermaid `tickInterval` in the code if needed

## See Also

- [spread-periodic-jobs](../spread-periodic-jobs/) - Tool to optimize job schedules
- [Mermaid Gantt Documentation](https://mermaid.js.org/syntax/gantt.html)
- [Prow Periodic Jobs](https://docs.prow.k8s.io/docs/components/core/crier/)

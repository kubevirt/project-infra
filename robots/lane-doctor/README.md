# lane-doctor

CLI tool to diagnose and remediate stuck Prow lane statuses on open PRs.

## Background

During lane migrations (e.g. bumping to k8s 1.36), the statusreconciler can
post bare `pending` GitHub commit statuses (no `target_url`, no actual prowjob)
on open PRs. When the affected lane is a required status check, those PRs are
blocked from merging. This tool automates the detection, triage, and
remediation workflow.

## Commands

### scan

Scans all open PRs for a given lane and classifies each as
stuck/missing/running/success/failed. Outputs a YAML report.

```bash
lane-doctor scan --lane pull-kubevirt-e2e-k8s-1.36-sig-compute-migrations -o scan.yaml
```

### prioritize

Reads a scan report and groups stuck PRs into priority tiers (P1–P4) based on
merge-readiness labels (`lgtm`, `approved`, hold status, draft).

```bash
lane-doctor prioritize -i scan.yaml -o priority.yaml
```

### trigger

Posts `/test <lane>` comments on PRs from a priority report in configurable
batches with wait intervals between them.

```bash
# Preview what would happen
lane-doctor trigger -i priority.yaml --group P1 --batch-size 10 --dry-run

# Run interactively (press Enter between batches)
lane-doctor trigger -i priority.yaml --group P1 --batch-size 10

# Run unattended with 4h waits between batches
lane-doctor trigger -i priority.yaml --yes --batch-wait 4h --batch-size 10
```

## Authentication

Provide a GitHub token via `--token-path` (path to a file) or the
`GITHUB_TOKEN` environment variable. The tool uses the Prow GitHub client
which supports both personal access tokens and GitHub App credentials.

## Global Flags

| Flag | Default | Description |
|---|---|---|
| `--repo` | `kubevirt/kubevirt` | GitHub repository in owner/repo format |
| `--token-path` | | Path to GitHub token file |
| `--dry-run` | `false` | Print actions without executing them |
| `--endpoint` | `https://api.github.com` | GitHub API endpoint |

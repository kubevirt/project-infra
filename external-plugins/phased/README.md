# phased - external plugin for Prow

## Motivation

Running expensive e2e tests before build and unit tests succeed, or while a PR
is still in review, wastes CI resources. The phased plugin supports running
presubmit jobs in successive phases as commit statuses succeed, as well as
requesting required manual jobs when a PR is ready for merging.

The [referee](../referee/) external plugin detects phased comments and skips
them when counting test request comments, to avoid skewing its retest metrics.

## Overview

Phased is a Prow external plugin with two independent execution paths:

- Status-driven phases: successful GitHub `status` events advance annotated
  presubmit jobs through their configured phases, without requiring review labels.
- Label-driven jobs: GitHub `pull_request` events (`labeled` and `synchronize`)
  request required manual presubmits when review labels allow it.

Both paths operate on `kubevirt/kubevirt` PRs targeting `main` or `master`, skip
draft, merged, closed, or explicitly non-mergeable PRs, and post GitHub comments
with `/test <job-name>` commands for Prow to execute. Prow and presubmit job
configs are fetched at runtime over HTTP from the configured `--prow-location`.

### Status-driven phases

Presubmit jobs opt in through the `phased.kubevirt.io/phase` annotation. Phases
are ordered numerically and need not be contiguous. Jobs without this annotation
do not participate; invalid integer values are logged and ignored.

Successful status events for annotated job contexts trigger evaluation; unrelated
contexts are ignored when the presubmit cache is available. The plugin finds
open PRs whose current HEAD SHA matches the event SHA and reads that SHA's
combined commit statuses. Every job in a phase must report `success` before the
next configured phase can run.
Missing, pending, failed, or error statuses prevent progression beyond that phase.

The plugin requests at most one next phase per evaluation. It only requests jobs
in that phase that have no status yet; jobs with any existing status are not
requested again. Status matching uses a job's explicit `context` when configured,
otherwise its `name`. The `/test` commands always use job names. For example:

```text
Phase 0 jobs completed, triggering phase 1 presubmits:
/test pull-example-e2e
```

The first configured phase must be started by another mechanism, typically
Prow's `always_run: true`. Later phases should use `always_run: false` so they
wait for the plugin rather than starting automatically.

### Recovery after restarts and missed events

At startup, phased scans all open `kubevirt/kubevirt` PRs using a fresh job
configuration snapshot. It evaluates the current HEAD statuses with the same
phase rules as status webhooks. If earlier phases succeeded, it requests the
missing jobs in the next phase directly; it does not rerun phase-0 jobs or replay
the label-driven path.

Recovery scans repeat ten minutes after the previous pass finishes, so changes
to phase annotations and missed webhook deliveries also recover without a
restart. Set `--reconcile-interval=0` for startup-only recovery, or use another
nonnegative duration to change the interval. Failed passes retry with exponential
backoff, starting at one second and capped at one minute. A single PR failure
does not prevent processing other PRs.

The scan uses at most four concurrent status reads. After scanning, recovery
requests are paced at one PR per thirty seconds. HEAD, eligibility, and statuses
are rechecked before posting. Webhooks continue to be handled during recovery;
their requests are not paced by the recovery limiter.

Status webhooks and recovery share per-PR locking and recent-request tracking.
Generated comments include hidden markers identifying each requested job by
HEAD SHA and phase. For ten minutes, markers in comments from the plugin's bot
account suppress duplicate requests, including after a restart. An in-memory
ledger covers delays before newly posted comments become visible through the
GitHub cache. Jobs with any existing status are never requested again by this
path, even if that status is a failure.

Scan and completion logs include PR counts for skipped, blocked, candidate,
requested, already requested, and failed evaluations. A request without statuses
becomes eligible again after the ten-minute grace period; monitor repeated
requests as a possible Prow dispatch or reporting problem.

### Label-driven jobs

The existing label-driven behavior remains available:

- On `labeled`: triggers when `lgtm` + `approved` are both present, or when
  `skip-review` is added
- On `synchronize`: triggers only when the `skip-review` label is present
- Selects presubmit jobs that are non-optional, not always-run, and have no
  path-based conditions (i.e. jobs requiring manual `/test` triggering)
- Excludes all jobs with the `phased.kubevirt.io/phase` annotation; status-driven
  progression and recovery own these jobs, regardless of their current status

These requests retain the comment prefix `Required labels detected, running
phase 2 presubmits:`. Here, "phase 2" is a legacy name for unannotated required
manual jobs, not jobs annotated with phase `2`. Review labels cannot restart
annotated jobs or bypass earlier phases. This path still does not check existing
statuses for unannotated jobs.

The `skip-review` label is restricted to `kubevirt-bot` and
`kubevirt-commenter-bot` via Prow label restrictions in `plugins.yaml`,
allowing automated PRs to trigger phase 2 without human review labels.

## Configuration

The plugin is registered in:

- `github/ci/prow-deploy/kustom/base/configs/current/plugins/plugins.yaml`

The external plugin registration must subscribe to both webhook types:

```yaml
external_plugins:
  kubevirt/kubevirt:
  - name: phased
    events:
    - pull_request
    - status
```

Add phase annotations to the relevant presubmits. For example, these fragments
define a build phase followed by e2e tests:

```yaml
presubmits:
  kubevirt/kubevirt:
  - name: pull-example-build
    always_run: true
    annotations:
      phased.kubevirt.io/phase: "0"
  - name: pull-example-e2e
    always_run: false
    annotations:
      phased.kubevirt.io/phase: "1"
```

Only annotate jobs that will report statuses for every applicable PR. The
status-driven path does not filter annotated jobs by branch/path conditions,
optionality, or `always_run`; a job that never reports a status blocks progression.

Deploy the N-phase plugin implementation and enable `status` events before
switching later-phase jobs to `always_run: false`, to avoid losing automatic
test coverage. The companion configuration change is
[project-infra#5262](https://github.com/kubevirt/project-infra/pull/5262).

Deployment manifests:

- `github/ci/prow-deploy/kustom/base/manifests/local/prow-phased-deployment.yaml`
- `github/ci/prow-deploy/kustom/base/manifests/local/prow-phased-service.yaml`

## Limitations

- Hardcoded to `kubevirt/kubevirt` in the source, not just a configuration choice
- Only triggers for PRs targeting `main` or `master` branches
- Does not handle `opened` events; PRs created with `skip-review` already applied
  will not trigger phase 2 until a subsequent `synchronize` event
- Status-driven progression handles commit `status` events, not `check_run` or
  `check_suite` events, and ignores non-success events.
- Status processing caches presubmit configuration and phased status contexts
  in memory for five minutes. Recovery bypasses this cache once per pass.
  Label-driven processing loads config separately.
- Comment-based triggering is not exactly-once delivery. The per-PR locks are
  process-local, so overlapping plugin replicas can race before either sees the
  other's request. Requests without statuses can be repeated after the grace
  period. Run one replica; restart markers reduce but do not eliminate this race.
- Shutdown cancels pacing and comment requests. Existing GitHub read operations
  without context support must finish before an in-flight scan can stop.

## Development

Build the binary:

```bash
make -f external-plugins/Makefile build PROW_PLUGIN="phased"
```

Run tests:

```bash
make -f external-plugins/Makefile test PROW_PLUGIN="phased"
```

Push the image:

```bash
make -f external-plugins/Makefile push PROW_PLUGIN="phased"
```

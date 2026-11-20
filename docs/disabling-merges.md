# Disabling Tide merges

Research for [kubevirt/project-infra#5342](https://github.com/kubevirt/project-infra/issues/5342).

Use this when a branch must stop accepting merges even though pull requests are otherwise approved. A required job that is already red, such as `pull-kubevirt-unit-test-s390x` on `main`, already keeps those pull requests out of the merge pool. These mechanisms are for a hard stop: code freeze, a flaky required job, or the window after a required job is made optional. `/override` satisfies a failing context and does not lift any of the blocks below.

Tide merges a pull request when it matches **any** query. A freeze has to close every query that still matches the target branch.

Today, `kubevirt/kubevirt` reaches Tide through five queries in [`config.yaml`](../github/ci/prow-deploy/kustom/base/configs/current/config/config.yaml):

| Query | What it merges |
| --- | --- |
| `lgtm` + `approved` + `approved-vep`, `includedBranches: [main]` | Approved VEP pull requests into `main` |
| `lgtm` + `approved` + `kind/bug` + `priority/critical-urgent`, `includedBranches: [main]` | Critical bugfixes into `main` |
| `lgtm` + `approved` + `kind/flake` + `priority/critical-urgent`, `includedBranches: [main]` | Critical flake fixes into `main` |
| `lgtm` + `approved`, no branch filter | Every other approved pull request, including `main` and release branches |
| `author: kubevirt-bot` + `skip-review`, no branch filter | Bot pull requests into every branch of `kubevirt/kubevirt` and several other repos |

The `tide.priority` list only orders the pool. `kind/failing-test` is on that list and is not a merge exception. There is no `kind/bugfix` label; the live exception labels are `kind/bug` and `kind/flake`.

`tide.sync_period` is `2m`, so a config or blocker change takes effect on the next sync.

## Recommendation

For a broken lane or a short freeze on one branch, use **tide/merge-blocker** (variant 3) once it is configured. Opening and closing one issue is the whole freeze. Put `branch:main` in the issue title so release branches keep merging.

That path is not wired up yet. Until the one-time config below lands, use a Tide-query change (variant 1 or 2). The query change is the only method that works with the current deployment.

Use the blockade plugin (variant 4) for a path freeze, such as "release branch accepts only docs". It is a poor fit for "this lane is broken, stop the branch".

## 1. Disable every Tide query for one branch

**Bandwidth:** medium. One `project-infra` pull request to start the freeze, and another to lift it. Each needs a second person because Tide merges `project-infra` itself. Freezing `kubevirt/kubevirt` leaves the `project-infra` queries in place, so the revert can still merge.

**Already possible:** yes. The general `kubevirt/kubevirt` query already has the code-freeze comment.

To stop every merge into `main`:

1. Remove the three queries that set `includedBranches: [main]` (`approved-vep`, `kind/bug`, `kind/flake`).
2. Uncomment the code freeze on the remaining `kubevirt/kubevirt` query:

```yaml
  - repos:
    - kubevirt/kubevirt
    excludedBranches:
    - main
    labels:
    - lgtm
    - approved
    missingLabels:
    - do-not-merge
    - do-not-merge/hold
    - do-not-merge/invalid-owners-file
    - do-not-merge/work-in-progress
    - do-not-merge/release-note-label-needed
    - do-not-merge/invalid-commit-message
    - needs-rebase
    - "dco-signoff: no"
```

3. Give `kubevirt/kubevirt` its own `skip-review` query with the same `excludedBranches`, and drop it from the shared `author: kubevirt-bot` query. `excludedBranches` applies to every repo in that query. Adding it in place would also stop bot merges to `main` of `project-infra`, `kubevirtci`, `containerized-data-importer`, `kubevirt-aie`, and `kubevirt-aie-webhook`.

Release branches keep merging through the general query. Repeat the same shape with `release-1.7` (or whichever branch) to freeze a release branch instead of `main`.

## 2. Same freeze, with critical exceptions

**Bandwidth:** medium, same as variant 1.

**Already possible:** yes. This is the code freeze the current queries were shaped for.

Do steps 2 and 3 from variant 1, and remove only the `approved-vep` query for `main`. Leave the two `includedBranches: [main]` queries that require both `priority/critical-urgent` and either `kind/bug` or `kind/flake`.

A pull request needs both labels. `priority/critical-urgent` on its own stays out of the pool. `approved-vep` pulls stay out too; keep that query only if VEP work should continue during the freeze. `skip-review` bot pulls still need the same exclusion as in variant 1, otherwise they skip the exception labels and merge anyway.

## 3. `tide/merge-blocker`

**Bandwidth:** low, after a one-time config change. Opening an issue starts the freeze. Closing it, or removing the label, ends it.

**Already possible:** no.

Tide only honors blocker issues when `tide.blocker_label` is set. KubeVirt's Tide config does not set it. Upstream Kubernetes sets `blocker_label: tide/merge-blocker` ([test-infra config](https://github.com/kubernetes/test-infra/blob/d9b63ee450b630ce64602d10303f6279729f42dc/config/prow/config.yaml#L925)). The behavior is documented at [Configuring Tide](https://docs.prow.k8s.io/docs/components/core/tide/config/#merge-blocker-issues) and implemented in [`pkg/tide/blockers`](https://github.com/kubernetes-sigs/prow/blob/main/pkg/tide/blockers/blockers.go).

Tide searches for **open issues** (not pull requests) with that label. An issue blocks only the repository it was filed in:

- A title with no `branch:<name>` token blocks every branch of that repo.
- Each `branch:<name>` token blocks that branch. Tokens can be repeated, and `branch:"release-1.7"` is accepted. The matcher is `(?im)\bbranch:[^\w-]*([\w-./]+)\b`.

An issue in `kubevirt/kubevirt` does not block `project-infra` or any other repo. The warning on the research issue overstates the blast radius.

The label `tide/merge-blocker` exists only on `kubevirt/hyperconverged-cluster-operator`, with `target: prs` and `addedBy: openshift-ci`, described as OpenShift CI compatibility. Tide ignores pull-request labels, so that entry does not enable this feature.

### One-time setup

In `tide:`:

```yaml
tide:
  blocker_label: tide/merge-blocker
```

On each repo that should be able to freeze itself, in [`labels.yaml`](../github/ci/prow-deploy/kustom/base/configs/current/labels/labels.yaml) (description must stay within 100 characters):

```yaml
- name: tide/merge-blocker
  color: e11d21
  description: Blocks Tide merges while this issue is open. Use branch:<name> in the title to scope it.
  target: issues
  prowPlugin: label
  addedBy: org members
```

Restrict who can apply it. The label plugin lets any commenter add a label that is not listed under `label.restricted_labels`, and these repos are public. Follow the `approved-vep` entry in [`plugins.yaml`](../github/ci/prow-deploy/kustom/base/configs/current/plugins/plugins.yaml):

```yaml
label:
  restricted_labels:
    kubevirt:
    - allowed_teams:
      - restricted-labels-kubevirt
      label: tide/merge-blocker
```

Label sync runs from a daily CronJob. The label has to exist on the repo before `/label tide/merge-blocker` works. See [labels-management.md](labels-management.md).

### Using it

File an issue in the repo to freeze. Example title:

```text
branch:main pull-kubevirt-unit-test-s390x is failing
```

Then, from an allowed user:

```text
/label tide/merge-blocker
```

Tide reports the blocking issue on the PR status and skips merges to the selected branches. Closing the issue or commenting `/remove-label tide/merge-blocker` lifts the freeze on the next sync.

Leaving `branch:main` out of the title also stops release-branch merges for that repo.

## 4. Blockade plugin

**Bandwidth:** medium. The blockade is config, so each change is a pull request in both directions.

**Already possible:** no. The plugin is not enabled, `plugins.blockades` is empty, and Tide does not list `do-not-merge/blocked-paths` in any `missingLabels`.

The plugin adds `do-not-merge/blocked-paths` when a pull request changes a protected path. It does not talk to Tide. Tide treats `missingLabels` as exact GitHub label matches, so an existing `do-not-merge` entry does not cover `do-not-merge/blocked-paths`. Without that missing label, Tide still merges the pull request.

A file is blocked when it matches a `blockregexps` entry and matches none of the `exceptionregexps`. The pull request is blocked when any changed file is blocked. Expressions are unanchored Go regular expressions: write `^docs/` rather than `docs/`, or `pkg/docs/foo` matches too.

The plugin runs on `opened`, `reopened`, and `synchronize` only. Pull requests that are already open keep merging until the next push. That makes it a weak emergency brake.

The sample in the issue does not parse. `exceptionregexps` has to be a nested list, and the plugin has to be enabled on the repo (org plugins and repo plugins are additive). A working path freeze looks like this:

```yaml
# plugins.yaml, next to the other plugin lists
plugins:
  kubevirt/kubevirt:
    plugins:
    - blockade

# plugins.yaml, top level, next to approve/lgtm
blockades:
- repos:
  - kubevirt/kubevirt
  branchregexp: "^main$"
  blockregexps:
  - ".*"
  exceptionregexps:
  - "^docs/"
  - "^tests/"
  explanation: "Merges to main are limited to docs and tests while pull-kubevirt-unit-test-s390x is broken."
```

Also add `do-not-merge/blocked-paths` to [`labels.yaml`](../github/ci/prow-deploy/kustom/base/configs/current/labels/labels.yaml) and to the `missingLabels` of every Tide query that should honor it, including the `skip-review` query. The label string is [`labels.BlockedPaths`](https://github.com/kubernetes-sigs/prow/blob/main/pkg/plugins/blockade/blockade.go).

This matches "freeze `main`, but still merge docs and tests". It does not match "stop this branch because a lane is red": test-only pulls would still merge, and untouched open pulls would not be labeled.

## Do not confuse these with a merge freeze

`release-blocker` (the external plugin and the `release-blocker/<branch>` labels) tells the release tool to hold a tag or release candidate. It does not remove pull requests from Tide.

`/hold` blocks one pull request. It does not freeze a branch.

## Follow-up

The research is done. The remaining work is the one-time variant 3 setup: `blocker_label`, an issue label on the repos that need it, and a `restricted_labels` entry. That pull request should not open a blocker issue.

# KubeVirt Release Operations for SIG-CI

This guide collects the release-related handoffs for the SIG-CI operations
group: what people must coordinate or verify, what automation already does,
and where the detailed procedures live. Use it alongside the
[release tracker][release-tracker], not instead of the release procedure.

The inventory was checked on 2026-11-04 against project-infra `main` at
`f14d1d360` and local copies of the linked repositories. It describes
checked-in automation, not confirmation that deployed jobs have succeeded.
Recheck the linked configuration before each release.

## Scope and Ownership

[CI operations](ci-operations-group.md#responsibilities-of-the-group) keeps
infrastructure operational and PRs unblocked, supports Prow configuration,
and routes non-infrastructure failures to the responsible SIG.
[SIG-buildsystem][buildsystem-charter] owns build and lane-progression
automation; lane stakeholders assess test signal and non-standard lanes.
SIG-release owns release timing and release decisions.

Branch/tag creation, signing, RC promotion, release notes, announcements,
API compatibility testdata, and the post-CDI-GA CDI pin are assigned in the
[release procedure][release-procedure] and
[release checklist][release-checklist]. They are not additional SIG-CI
manual publication tasks. Support their pipelines when infrastructure fails.

The important distinction is the event, not just the release date:

- **New minor cycle/branch:** coordinate lane readiness, merge-policy
  transitions, and release-branch CI onboarding, mostly before GA.
- **Every release tag, including patch and prerelease tags:** observe
  publication and generated support-matrix updates. Existing branch CI is
  reused; a patch does not normally need a new job file.
- **After GA:** verify outstanding proposals and reports, and maintain
  release CI as needed. Provider rotation and branch retirement are separate,
  conditional workflows, not mandatory cleanup after every patch.

## Before Code Freeze: Providers and Lane Readiness

The [release tracker][release-tracker] and each release's
[schedule][release-schedules] set provider, voting-lane, mandatory-presubmit,
and non-standard-lane milestones. Use those dates rather than assuming the
work starts on GA day.

### Existing Provider and Lane Automation

The [Kubernetes automation overview][k8s-automation] and
[project-infra periodic configuration][infra-periodics] describe the chain:

- `periodic-kubevirtci-cluster-minorversion-updater` proposes new providers;
  `periodic-kubevirtci-cluster-patchversion-updater` proposes patch updates.
- `periodic-kubevirtci-provider-presubmit-creator` proposes provider build
  presubmits; `periodic-kubevirtci-bump-kubevirt` proposes integration into
  KubeVirt.
- `periodic-kubevirt-job-copier` proposes SIG compute, network, storage, and
  operator lanes for the newest provider. New presubmits start on-demand
  and nonblocking: `always_run: false`, `optional: true`.
- `periodic-kubevirt-presubmit-requirer` runs monthly and proposes lane
  progression and old-provider `always_run` changes. The
  [implementation][require-presubmits] first enables execution, then makes
  the latest lanes required (`optional: false`); it also adjusts previous
  versions' pre-merge execution.
- `periodic-kubevirt-job-remover`,
  `periodic-kubevirtci-cluster-unsupported-version-remover`, and
  `periodic-kubevirtci-provider-presubmit-remover` propose old-lane/provider
  removal, with usage/readiness checks.

These are scheduled proposal mechanisms, not proof of readiness or that
the changes have merged. They follow Kubernetes/provider availability;
they do not rerun an entire migration for every KubeVirt patch tag.

### Human Handoff

- [ ] Find the generated PRs and verify the provider is published,
      integrated into KubeVirt, and actually usable by the intended lanes.
- [ ] Coordinate signal review and promotion with SIG-buildsystem and
      affected SIGs. Do not rely on the monthly job meeting a release deadline.
- [ ] Confirm stakeholders have handled migrations, multus, IPv6, and
      other non-standard lanes; the standard copier does not cover everything.
- [ ] Coordinate changes to required contexts using the
      [lane-migration process](lane-migration-process.md) and
      [migratestatus](migratestatus.md). Hold approved migration PRs until an
      operator can manage the transition and avoid a statusreconciler-triggered
      burst across open PRs.
- [ ] Before deleting a provider, audit versioned release job files as well
      as `main`. The remover jobs' configured usage checks point at the
      unversioned KubeVirt job files, not all release branches.

**Migration safety caveat:** the linked migration guide says to scale down
statusreconciler, but shows `--replicas=1` and does not document restoration.
Do not copy that command blindly. Check the running deployment and current
operator procedure, record the original replica count, arrange suspension
and restoration explicitly, and dry-run status retirement before mutation.

## Code Freeze and New Minor Branch

### Merge Policy

- [ ] With SIG-release, confirm when freeze and thaw apply and which
      branches or exceptions they affect. Implement/review the corresponding
      Tide queries in [Prow configuration][prow-config].
- [ ] Verify both ordinary and exceptional merge routes after the change;
      thaw is not triggered automatically by a release tag. Do not assume the
      freeze exclusion is still enabled in the checked-in configuration.

### Generated Release Presubmits

The [release tool][release-tool] calls `forkProwJobs()` when cutting a branch
unless Prow handling is skipped. It runs `config-forker` on
`kubevirt-presubmits.yaml`, writes `kubevirt-presubmits-X.Y.yaml`, and opens
a project-infra PR. `configureReleaseJob()` converts `run_before_merge`
jobs to `always_run` and removes the Bazel-cache preset. An existing output
file is not regenerated by this path.

This is automation invoked by release management, not a background branch
discovery service. It generates presubmits, not a complete fork of
periodics, postsubmits, or dashboards. The tracker item "Create
release-1.XX prow config" is an outcome to verify, not a direction to copy
all jobs by hand.

- [ ] Locate and review the generated PR with the release manager. If
      generation was skipped or failed, agree on recovery rather than creating
      a competing configuration independently.
- [ ] Inspect `fork-per-release` annotations, branch filters, contexts,
      `always_run`, `optional`, and `run_before_merge`. Ensure main's jobs cannot
      also match the release branch with the same context. A versioned job
      **name** does not guarantee a unique status **context**.
- [ ] Review provider targets, runner images, Go/toolchain settings,
      credentials/presets, architecture, and job commands for compatibility
      with the release branch. The generator currently passes Go `1.24` to
      `config-forker`; generated output is not automatically release-ready.
- [ ] Merge the reviewed configuration, verify Prow has loaded it, and
      check the new branch's intended required checks after branchprotector
      reconciliation. Inspect an actual release-branch PR's routing and jobs.
- [ ] Verify dashboard/report coverage and initial job results. Record the
      configuration PR and any unresolved infrastructure work in the release
      tracker; route product/test failures to the owning SIG.

Use the [job configuration directory][kubevirt-jobs] for current examples.
For changes, require `check-prow-config` and applicable TestGrid checks.
The [project-infra presubmits][infra-presubmits] include a KubeVirt
presubmit checker, but it targets main's unversioned file, so it is not a
substitute for release-file review. Authorized contributors can use
[/rehearse][rehearse] for supported presubmits; it does not rehearse
periodics or postsubmits.

## Release Tags: RC, GA, and Patch Verification

The [KubeVirt postsubmits][kubevirt-postsubmits] match SemVer tags, including
prereleases, for these two jobs:

### Release Publication

`push-release-kubevirt-tag` invokes the tagged branch's
[`automation/release.sh`][release-automation]. The current script builds
and verifies artifacts, pushes product images, uploads GitHub release
assets (including conformance/testing manifests), uploads testing manifests
to GCS, generates `stable.txt`, and invokes API-reference and staging
publication. Older release branches may contain older script behavior.

- [ ] Check the tag job and relevant published outputs with the release
      manager. A tag's existence or a successful build phase alone does not
      prove that all later upload/publication phases completed.
- [ ] Diagnose infrastructure failures in credentials, registries, signing
      access, storage, or CI capacity; coordinate recovery/re-execution with
      release management. Do not change tags or publication decisions as an
      infrastructure workaround.
- [ ] For GA/patch releases, check that stable-version pointers and
      consumer-visible artifacts are appropriate; for RCs, do not expect the
      stable pointer to advance to the RC.

The current script selects the global `stable.txt` by version-sorting release
names after excluding `-rc`, `-alpha`, and `-beta`; it does not gate selection
on release-manager approval or the GitHub prerelease flag. A new highest-version
GA/patch can advance it while still marked prerelease and before later
publication phases finish. An older-minor patch need not advance it. Verify
these outcomes separately from release promotion and whole-job success.

Do not recreate the old per-release testing-manifest periodics: the current
script explicitly replaces them. Release promotion, documentation content,
and announcements remain release-manager/Docs/Community decisions.

### Support Matrix Proposal

`push-kubevirt-update-support-matrix` runs `robots/kubevirt get
support-matrix` and proposes an update to
[`sig-release/releases/k8s-support-matrix.md`][support-matrix]. It reads
project-infra `main`, not the tag's historical infrastructure snapshot.

- [ ] Verify the job succeeded and review/route the resulting PR. Merge
      the release-branch job configuration before expecting the matrix to
      include it; investigate ordering or generation failures if it is missing.
- [ ] Check the proposed combinations against the intended tested support
      window with SIG-release. The [generator][matrix-generator] derives data
      from required, `always_run` compute jobs in versioned presubmit files and
      Kubernetes schedule/EOL data. Context naming and execution flags affect
      the result; the matrix is not independent evidence of passing tests.

Patch releases use the same minor-branch CI configuration. There is no
general per-patch requirement to copy lanes, bump all CI images, create a
TestGrid dashboard, or retire another release branch.

## After GA: Reports and Ongoing Branch Care

### Dashboards and Lane Overviews

- [ ] Check the new branch's jobs are discoverable through the appropriate
      dashboards and [CI reports](reports.md). TestGrid membership comes from
      job annotations and [TestGrid configuration][testgrid-docs], not from the
      mere existence of a release tag. A separate dashboard per release is not
      required by the documented process.
- [ ] If dashboard membership changes, review declarations/groups in
      `github/ci/testgrid/gen-config.yaml` and job `testgrid-dashboards`
      annotations. `post-project-infra-update-testgrid-config` generates and
      uploads the configuration after relevant config changes merge.
- [ ] Verify release-branch HTML lane overviews are fresh.
      `post-project-infra-kubevirt-e2e-jobs-overview` generates main and the
      intended latest three release overviews, but its trigger only matches
      unversioned presubmit/periodic filenames. A release-file-only change
      does not trigger regeneration. Its reverse lexical filename sort is
      also not semantic version order (for example, `1.9` versus `1.10`).
      Investigate missing/stale output rather than treating it as absent CI.

Both publication jobs are defined in
[project-infra postsubmits][infra-postsubmits].

### Performance Release Graphs

- [ ] With the performance stakeholders, arrange the release benchmark
      report when required by the release tracker. Update release version,
      start date, and release/Kubernetes markers in
      [`robots/perf-report-creator/shape.yaml`][performance-shape] using the
      [performance-report instructions][performance-report].
- [ ] After that change merges, verify
      `post-project-infra-kubevirt-releasegraph` and the generated PR to
      `kubevirt/ci-performance-benchmarks`; arrange review/publication with the
      stakeholders. This workflow requires manual input, but does not establish
      a separate report obligation for every patch release.

### Images, Upgrades, and Backports

- [ ] When release CI needs a tooling fix, selectively update compatible
      runner images in its versioned job file. The normal
      [image bumper][image-bumper] excludes versioned release files; so does
      [`hack/update-jobs-with-latest-image.sh`][image-helper]. Do not assume
      automated main-branch bumps maintain release CI, or blanket-bump old
      branches just because a release was published.
- [ ] When checking upgrade signal, inspect the selected previous-version
      baseline and explicit overrides before proposing a bump. KubeVirt's
      [upgrade-version detection][upgrade-version] selects previous releases
      dynamically; a manual baseline edit is not inherently a post-GA task.
- [ ] Preserve working unit/functional lanes for branches accepting
      backports, as required by [backport policy][backport-policy]. Support
      cherrypicker/release-blocker infrastructure if it fails, without taking
      over backport approval or release-blocking decisions.

## Conditional Retirement, Not Automatic Post-Release Cleanup

Do not infer EOL from a new tag or delete the oldest branch on every GA.
Agree the support/testing decision with SIG-release first, using the
[release lifecycle][release-policy], [compatibility documentation][compatibility],
and [support matrix][support-matrix]. Compatibility claims, actively tested
combinations, provider availability, and branch archival are different things.

For an explicitly approved archival operation:

1. Audit remaining users, release job/provider references, dashboards, and
   required contexts; agree what CI and backport support will end.
2. Review removal of obsolete release presubmit files and mark the selected
   branches `unmanaged` in [branchprotector configuration][prow-config], so
   reconciliation does not overwrite archival protection.
3. After merge, verify that [branchprotector][branchprotector] has completed
   a reconciliation cycle (configured hourly at minute 54).
4. An authorized administrator can then apply the agreed branch lock and
   verify the resulting protection and merge behavior.

[`hack/archive-release-branches.sh`][archive-script] implements a historical
archival operation. It requires admin access, defaults to a fixed legacy
branch list, and replaces protection settings, including clearing required
checks/review restrictions. It is not a generic safe cleanup command. Review
its implementation and pass only an explicitly approved branch list if reused.

## Scattered Sources and Consolidation Follow-Ups

This document consolidates the operational handoffs, not the implementation
or release policy. Keep detailed commands and policy in their owning sources:

- **Release tracking:** link this guide from the
  [release-tracker template][release-tracker] and future schedules instead
  of duplicating operations checklists. Track completion there; edit schedule
  YAML rather than only its generated Markdown.
- **Provider/lane lifecycle:** retain the [Kubernetes automation
  inventory][k8s-automation], [provider developer guide][provider-guide],
  [CI-provider policy][ci-provider-policy], and
  [robot reference][kubevirt-robot] as detailed sources. The robot README's
  `require presubmits` explanation contradicts the implementation about
  `optional`; correct that explanation centrally, not in every schedule.
- **Lane migration safety:** keep the executable process in
  [lane-migration-process.md](lane-migration-process.md) and
  [migratestatus.md](migratestatus.md). Resolve the replica-count discrepancy
  and document restoration before treating the former as a copy/paste runbook.
- **Reports and images:** keep TestGrid details in its
  [README][testgrid-docs], performance inputs in the
  [report-creator README][performance-report], and image exclusions in the
  [bumper README][image-bumper]. Fix the overview trigger/version sorting in
  automation rather than adding a permanent manual regeneration chore.
- **Support policy:** reconcile the compatibility document's fixed release
  compatibility wording with the release lifecycle's shrinking validation
  window. Record testing/EOL decisions centrally before automating retirement.
- **Discovery:** the [CI operations overview](ci-operations-group.md) links
  here. Cross-repository links above are consolidation recommendations;
  those repositories and the identified automation gaps are not changed by
  adding this guide.

[archive-script]: ../hack/archive-release-branches.sh
[backport-policy]: https://github.com/kubevirt/kubevirt/blob/main/docs/release-branch-backporting.md
[branchprotector]: ../github/ci/prow-deploy/kustom/base/manifests/local/branch-protector.yaml
[buildsystem-charter]: https://github.com/kubevirt/community/blob/main/sig-buildsystem/charter.md
[ci-provider-policy]: https://github.com/kubevirt/kubevirt/blob/main/docs/release.md#ci-provider
[compatibility]: https://github.com/kubevirt/kubevirt/blob/main/docs/kubernetes-compatibility.md
[image-bumper]: ../robots/image-bumper/README.md
[image-helper]: ../hack/update-jobs-with-latest-image.sh
[infra-periodics]: ../github/ci/prow-deploy/files/jobs/kubevirt/project-infra/project-infra-periodics.yaml
[infra-postsubmits]: ../github/ci/prow-deploy/files/jobs/kubevirt/project-infra/project-infra-postsubmits.yaml
[infra-presubmits]: ../github/ci/prow-deploy/files/jobs/kubevirt/project-infra/project-infra-presubmits.yaml
[k8s-automation]: https://github.com/kubevirt/kubevirtci/blob/main/cluster-up/cluster/K8S_AUTOMATION.md
[kubevirt-jobs]: ../github/ci/prow-deploy/files/jobs/kubevirt/kubevirt
[kubevirt-postsubmits]: ../github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-postsubmits.yaml
[kubevirt-robot]: ../robots/kubevirt/README.md
[matrix-generator]: ../pkg/kubevirt/cmd/get/support-matrix.go
[performance-report]: ../robots/perf-report-creator/README.md#performance-benchmark-release-graphs
[performance-shape]: ../robots/perf-report-creator/shape.yaml
[provider-guide]: https://github.com/kubevirt/kubevirtci/blob/main/cluster-up/cluster/K8S_DEV_GUIDE.md
[prow-config]: ../github/ci/prow-deploy/kustom/base/configs/current/config/config.yaml
[rehearse]: ../external-plugins/rehearse/README.md
[release-automation]: https://github.com/kubevirt/kubevirt/blob/main/automation/release.sh
[release-checklist]: https://github.com/kubevirt/sig-release/blob/main/releases/release-checklist.md
[release-policy]: https://github.com/kubevirt/kubevirt/blob/main/docs/release.md
[release-procedure]: https://github.com/kubevirt/kubevirt/blob/main/docs/release-procedure.md
[release-schedules]: https://github.com/kubevirt/sig-release/tree/main/releases
[release-tool]: ../releng/release-tool/release-tool.go
[release-tracker]: https://github.com/kubevirt/sig-release/blob/main/.github/ISSUE_TEMPLATE/release-tracker.md
[require-presubmits]: ../pkg/kubevirt/cmd/require/presubmits.go
[support-matrix]: https://github.com/kubevirt/sig-release/blob/main/releases/k8s-support-matrix.md
[testgrid-docs]: ../github/ci/testgrid/README.md
[upgrade-version]: https://github.com/kubevirt/kubevirt/blob/main/tests/operator/version/version.go

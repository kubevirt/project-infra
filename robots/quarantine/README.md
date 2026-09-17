# quarantine

Looks up KubeVirt Ginkgo specs, reports flake candidates, and rewrites test source so CI treats a spec as quarantined.

There is no published image; CI uses `go run ./robots/quarantine …` from a project-infra checkout. `--test-source-path` and `--job-config-path` are relative to the process working directory (project-infra root in Prow).

## How a spec is quarantined

`pkg/ginkgo.QuarantineTest` (used by `test` and `auto`):

1. Ginkgo dry-run on `--test-source-path` to find the spec.
2. In the leaf file, replace the first quoted leaf-node text `"…"` with `"[QUARANTINE]…", decorators.Quarantine`.
3. Ensure `import "kubevirt.io/kubevirt/tests/decorators"`, then `goimports`.

`decorators.Quarantine` is KubeVirt’s alias for `Label("QUARANTINE")`. Docs: https://github.com/kubevirt/kubevirt/blob/main/docs/quarantine.md#quarantine-pr

`--test-name` matching (`pkg/ginkgo.ByName`): the string must contain the leaf text and every container/Describe text (whitespace-normalized; Ginkgo `When` nodes also match without their extra `when ` prefix). Use the full Ginkgo spec string.

Shared flag:

| Flag | Default |
| --- | --- |
| `--test-source-path` | `../kubevirt/tests/` |

## `quarantine test`

Quarantine one spec in place.

```bash
go run ./robots/quarantine test \
  --test-name="<full spec text>" \
  --test-source-path=/path/to/kubevirt/tests
```

`--test-name` must be non-empty at lookup time (flag default is `""`).

## `quarantine info`

Print the matching `types.SpecReport` as JSON (file, line, labels, node hierarchy). No files are modified.

```bash
go run ./robots/quarantine info --test-name="<full spec text>" --test-source-path=/path/to/kubevirt/tests
```

## `quarantine report`

HTML report of flake candidates. Does **not** edit tests.

Data:

1. Aggregate 24h flakefinder JSON for `kubevirt/kubevirt` (`pkg/flake-stats`; org/repo not flag-overridable here).
2. For each test, scrape search.ci for **72h and 336h** windows (`excludeName=periodic-.*`). Keep impacts at **≥20%** (72h) or **≥5%** (336h).
3. Drop remaining impacts that are rehearsal / `check-tests-for-flakes` / `check-dequarantine-test` URLs, lanes that were not in the flake-stats failure set, or “clustered” runs (every listed build’s `junit.functest.xml` has no suite with fewer than 5 failures).
4. Group by `[sig-…]` in the test name (`NONE` if absent). Required vs optional lanes come from the job config (`ContextRequired()`: `optional: false` and not `skip_report`). Optional and release-branch lanes stay in the HTML; the page hides them until **Show optional lanes** / **Show release branch lanes** is checked.

`HasRecentFailures` is display-only on this command (does not drop rows).

```bash
go run ./robots/quarantine report [flags]
```

| Flag | Default | Notes |
| --- | --- | --- |
| `--days-in-the-past` | `14` | Flakefinder days (yesterday backward). |
| `--output-file` | temp `most-flaky-tests-*.html` | |
| `--overwrite-output-file` | `false` | Required if `--output-file` already exists. |
| `--filter-periodic-job-run-results` | `true` | When true, **exclude** flake-stats jobs whose names start with `periodic`. Search.ci scrape always uses `excludeName=periodic-.*` regardless. |
| `--filter-lane-regex` | `rehearsal` | Exclude flake-stats job names that match. |
| `--include-rolling-window` | `true` | Also fetch today’s rolling-window flakefinder JSON (warn and skip if missing). |
| `--max-failure-age` | `72h` | Recent-failure highlighting on search.ci builds. |
| `--min-recent-failures` | `2` | Minimum recent builds per impact for that highlight. |
| `--min-failure-interval` | `24h` | Span between oldest and newest recent failure must be ≥ this. |
| `--job-config-path` | `github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-presubmits.yaml` | Must contain required jobs. |
| `--job-config-org-repo` | `kubevirt/kubevirt` | Key in that YAML. |

Prow: `periodic-publish-kubevirt-most-flaky-tests-report` (`cron: '*/30 * * * *'`) runs `quarantine report --output-file /tmp/index.html` and copies to `gs://kubevirt-prow/reports/most-flaky-tests/kubevirt/kubevirt/`.

## `quarantine auto`

Selects candidates, quarantines source, writes a PR body. **No `--dry-run`.** Files are modified in place.

Unlike `report`: uses **only** the 336h search.ci window (≥5% impact); **does not** include today’s rolling-window flakefinder report; ignores tests named `AfterSuite`; flake-stats still drops jobs whose names start with `periodic` and keeps only lanes matching `--matching-lane-regex`; keeps a test only if at least one remaining impact is a **required** lane (`ContextRequired()`: `optional: false` and not `skip_report`) whose job history URL last path element matches `--matching-lane-regex`, and that impact passes the recent-failure filter. Shared search.ci filters from `report` (rehearsal / flake-check / clustered junit) still apply.

```bash
go run ./robots/quarantine auto \
  --test-source-path=/path/to/kubevirt/tests \
  --pr-description-output-file=/tmp/pr-description.md
```

| Flag | Default | Notes |
| --- | --- | --- |
| `--days-in-the-past` | `14` | |
| `--max-tests-to-quarantine` | `1` | `0` means no ceiling. |
| `--release-lane-suffix` | empty | Sprintf’d into `--matching-lane-regex` (main branch vs e.g. `-1.7`). |
| `--matching-lane-regex` | `^pull-.*-sig-(compute(-serial\|-migrations\|-arm64)?\|network\|storage\|operator)%s$` | Must contain a `%s` for the suffix. |
| `--max-failure-age` / `--min-recent-failures` / `--min-failure-interval` | `72h` / `2` / `24h` | Hard filter on impacts (not display-only). |
| `--job-config-path` / `--job-config-org-repo` | same defaults as `report` | Required-job set. |
| `--labels-yaml` | `github/ci/prow-deploy/kustom/base/configs/current/labels/labels.yaml` | Maps Ginkgo `sig-*` / `wg-*` labels to `/sig …` and `/wg …` in the PR body (aliases in `cmd/labels.go`). No matching label → `/sig compute`. |
| `--pr-description-output-file` | temp `pr-description-*.md` | Overwrite is allowed. Template: `cmd/auto-quarantine-pr-description.gomd`. |

If nothing qualifies, it logs `no tests to quarantine` and returns without editing tests or writing the PR description (`git-pr.sh` then no-ops).

Prow: `periodic-kubevirt-auto-quarantine` (`cron: 3 * * * *`, `prow-workloads`) runs `quarantine auto` inside `hack/git-pr.sh` against **kubevirt/kubevirt** (`--branch auto-quarantine`, `--labels approved,kind/auto-quarantine`, `--missing-labels lgtm,approved,do-not-merge/hold`, `--release-note-none`). Clones `kubevirt/kubevirt` as an extra_ref and passes `--test-source-path` at that checkout’s `tests/` directory.

## Development

```bash
go test ./robots/quarantine/...
```

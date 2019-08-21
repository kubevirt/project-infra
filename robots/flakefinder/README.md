flakefinder
===========

`flakefinder` is a tool to find flaky tests.

`flakefinder` does the following:

1. Report creation
  i. Fetching all merged PRs within a given time period
  ii. Getting the last commit of each PR which got merged
  iii. Correlate the PR with all prowjobs which were running against this last commit
2. creates/updates html document in GCS bucket /reports/flakefinder/
  i. flakefinder-$date.html - extract skipped/failed/success from the junit results and create a html table
  ii. index.html

Selecting the right builds:

- Filter out not merged PRs
- Filter out identical prow jobs on multiple PRs (can be because of the merge pool)
- Filter out jobs which don't have a junit result
- Only shows test results for all lanes where a test at least failed once on one of the found lanes
- Only take prow jobs into account which were run on the commit which got merged

Running flakefinder locally
---------------------------

Prerequisite is having set `GOOGLE_APPLICATION_CREDENTIALS` to location of a service account credentials file.


Running the flakefinder locally:

```bash
bazel run //robots/flakefinder -- -token=$HOME/.gh_access_token
```

Pushing a new image to public docker repository:

```bash
bazel run //robots/flakefinder:push --host_force_python=PY2
```


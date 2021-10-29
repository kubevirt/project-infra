flake-report-creator
====================

aims to use the flakefinder logic of creating reports for a set of GCS job directories, whereas it just fetches all subdirectories, tries to find junit files and returns a report of the aggregated data.

This enables a user to create an ad-hoc report of any GCS directories that contain kubevirt testing junit files, regardless of whether they actually are of the same type, in turn enabling a report of flaky tests over different job types or even just selected PRs. The user needs to make sure whether that makes sense of course 😏

Usage
-----

Example: we want to create a report from all job runs of two PRs on openshift-ci. We want to see the flakiness in a matrix over those two PRs. But we want to skip all jobs that don't match a certain regular expression.

```bash
$ bazel run //robots/cmd/flake-report-creator -- \
    --sub-dir-regex='.*-(e2e-[a-z\d]+)$' \
    --bucket-name=origin-ci-test \
    --job-data-path=pr-logs/pull/openshift_release/23021 \
    --job-data-path=pr-logs/pull/openshift_release/22352
...
2021/10/29 12:57:54 main.go:216: writing output file to /tmp/flakefinder-3764038013.html
```

Result: ![Report for example 1](./example_1.png)

# KubeVirt Project Infrastructure Tools

This repository provides supporting code for the project infrastructure. This is
what you can find on each of the repo directories

* `cni-plugins`: code to deploy CNI plugins (currently only sriov-passthrough-cni
is available)

* `docs`: extended documentation of several aspects of our infrastructure

* `external-plugins`: Prow plugins used on our setup

* `github/ci`: Infrastructure code for our main deployments:

  * `github/ci/prow-deploy`: Ansible code for testing and deploying Prow components,
  includes Prow configuration under github/ci/prow-deploy/files

  * `github/ci/services`: Code to manage additional CI services

  * `github/ci/testgrid`: Code to manage the configuration for our
  [testgrid setup](https://testgrid.k8s.io/kubevirt)

* `images`: Definition of container images used in CI

* `limiter`: Tool used to control connections for GCE buckets to the outside world
depending on billing alerts. See [README](limiter/README.md)

* `releng/release-tool`: Tool for creating KubeVirt releases

* `robots`: Automation tools

  * `robots/botreview`: Tool to automate reviews of repetitive PRs that are created through automation

  * `robots/ci-usage-exporter`: Prometheus exporter to expose CI infrastructure
  information

  * `robots/dependabot`: Tool to create PRs to resolve Github dependabot alerts in kubevirt org repositories

  * `robots/flakefinder`: Tool to create statistics from failed tests of PRs.
  See [README](robots/flakefinder/README.md)

  * `robots/flake-stats`: Provides a more condensed view on where flakes are causing the most impact. See [README](robots/flake-stats/README.md)

  * `robots/indexpagecreator`: Creates flakefinder index page

  * `robots/kubevirt`: Provides commands to manipulate the SIG jobs (periodic and presubmit) that are testing kubevirt with kubevirtci

    See [README](robots/kubevirt/README.md)

  * `robots/kubevirtci-bumper`: Tool to automatically bump kubevirtci providers.
  See [README](robots/kubevirtci-bumper/README.md)

  * `robots/kubevirtci-presubmit-creator`: Creates kubevirtci presubmit job
  definitions for new providers

  * `robots/labels-checker`: Checks whether a PR has certain labels

  * `robots/retests-to-merge`: Tool to check recent approved PRs for retest comments. See
  [README](robots/retests-to-merge/README.md)

  * `robots/uploader`: Tool to mirror bazel dependencies on GCS. See
  [README](robots/uploader/README.md)

## Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details on how to contribute.

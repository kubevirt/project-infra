# KubeVirt Project Infrastructure Tools

This repository provides supporting code for the project infrastructure.

 * cni-plugins

 Code to deploy CNI plugins (currently only sriov-passthrough-cni is available)

 * external-plugins

 Prow plugins used on our setup

 * github/ci

 Infrastructure code for ou main deployments:

   * github/ci/phx-prow

   Ansible code for provisioning phx-prow cluster

   * github/ci/prow-deploy

   Ansible code for testing and deploying Prow components, includes Prow configuration under github/ci/prow-deploy/files

   * github/ci/prow-workloads

   Ansible code for provisioning prow-workloads cluster

   * github/ci/services

   Code to manage additional CI services

   * github/ci/testgrid

   Code to manage the configuration for our [testgrid setup](https://testgrid.k8s.io/kubevirt)

 * images

 Definition of container images used in CI

 * limiter

 Tool used to control connections for GCE buckets to the outside world depending on billing alerts. See [README](limiter/README.md)

 * plugins/cmd/uploader

 Tool to mirror bazel dependencies on GCS. See [README](plugins/cmd/uploader/README.md)

 * releng/release-tool

 Tool for creating KubeVirt releases

 * robots

 Several Go based automation tools

   * robots/ci-usage-exporter

   Prometheus exporter to expose CI infrastructure information

   * robots/flakefinder

    Tool to create statistics from failed tests of PRs. See [README](robots/flakefinder/README.md)

   * robots/flake-issue-creator

   Tool to create kubevirt/kubevirt issues from flake test results

   * robots/indexpagecreator

   Creates flakefinder index page

   * robots/kubevirtci-bumper

   Tool to automatically bump kubevirtci providers. See [README](robots/kubevirtci-bumper/README.md)

   * robots/kubevirtci-presubmit-creator

   Creates kubevirtci presubmit job definitions for new providers

   * robots/kubevirt-presubmit-requirer

   Updates kubevirt presubmit jobs definitions to make required jobs related to new providers

## Contributing

Please see [CONTRIBUTING.md](CONTRIBUTING.md) for details on how to contribute.

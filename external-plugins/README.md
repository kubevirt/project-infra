# KubeVirt Prow in-house external plugins

Collection of KubeVirt CI Prow opinionated in-house external plugins

* [Coverage](./coverage/README.md): triggers Go unit test coverage ProwJobs on
  Pull Requests containing Go code changes

* [Phased](./phased/README.md): triggers phase 2 presubmit ProwJobs when a Pull
  Request is labeled ready for merging

* [Referee](./referee/README.md): enforces rules defined to keep KubeVirt CI
  healthy

* [Rehearse](./rehearse/README.md): triggers test runs of modified ProwJobs
  directly from a Pull Request

* [Release-blocker](./release-blocker/README.md): manage release-blocker labels
  on issues and Pull Requests for the [release-tool](../releng/release-tool).

* [Test-subset](./test-subset/README.md): triggers a targeted subset of e2e
  tests on a Pull Request

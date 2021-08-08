# CI maintainers

This document outlines the various responsibilities of the maintainer role in
KubeVirt CI. It is inspired by the [KubeVirt community membership document] and
the [Kubernetes community membership document].

KubeVirt CI maintainers are community members that can control CI
infrastructure.

## Requirements

* Be an [approver] of https://github.com/kubevirt/project-infra for at least 1
month.

* A reasoned nomination by a current CI maintainer, a GitHub user already
present in the [current CI maintainer] list. The nomination should be in the
form of a PR adding the new maintainer to the [current CI maintainer] list and
should include an explanation of the reasons to add the new maintainer in the
PR description.

## Responsibilities and privileges

* Contribute infrastructure code, like Prow configuration, automation code,
create or modify manifests to deploy additional services or changes in the
monitoring stack.
* Review and eventually approve infrastructure code proposed by others. For
example Prow configuration to add a new repo, new Prow job definitions or
tuning existing alerts.
* Have access to the CI clusters, user interface of the infrastructure providers,
the private monitoring dashboards and the channels reserved to receive
infrastructure notifications and alerts ([kubevirt-ci-monitoring] and
[kubevirt-ci-infra-monitoring]).
* Have access to automation secrets.
* Improve observability of the infrastructure.
* Design and implement alerts.
* Participate in infrastructure incident handling.
* Participate in post-mortems and contribute code to prevent detected flaws.


[KubeVirt community membership document]: https://github.com/kubevirt/community/blob/master/membership_policy.md
[Kubernetes community membership document]: https://github.com/kubernetes/community/blob/master/community-membership.md
[approver]: https://github.com/kubevirt/community/blob/master/membership_policy.md#approver
[current CI maintainer]: ../OWNERS_ALIASES
[kubevirt-ci-monitoring]: https://app.slack.com/client/T027F3GAJ/CTFN306KC
[kubevirt-ci-infra-monitoring]: https://app.slack.com/client/T027F3GAJ/C01MJUAT7GD

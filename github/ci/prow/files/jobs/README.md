# KubeVirt Prow Job Configs

Here are prow job configs for the KubeVirt prow deployment.

Please add jobs following this pattern: `[org/]repo/repo-*.yaml`.

Examples:
 * `myorg/myrepo/myrepo-presubmits.yaml`
 * `project-infra/project-infra-periodics.yaml`
 * `project-infra/project-infra-presubmits.yaml`
 * `kubevirtci/kubevirtci-postsubmits.yaml`

The file basename will be used as a configmap key so they need to be unique
across this directory.

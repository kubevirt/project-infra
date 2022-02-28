# Loki

Customization and deployment of [Loki] on Kubevirt CI cluster.
It uses internally these bazel [gitops rules].

**Note**

The manifests are based on the helm charts for [loki](https://github.com/grafana/helm-charts/tree/main/charts/loki) and [promtail](https://github.com/grafana/helm-charts/tree/main/charts/promtail).

## Deployment

You need:
* a kubernetes configuration at `~/.kube/config` with an user allowed to
create [these resources](./manifests).
* [bazelisk] installed.

Then, from the root of project-infra run:
```
$ ./github/ci/services/loki/hack/deploy.sh production-control-plane
```

## Tests

Can be tested locally using [kind] and [bazelisk], from the root of project-infra:
```
$ kind create cluster
$ ./github/ci/services/loki/hack/deploy.sh testing
$ bazelisk test //github/ci/services/loki/e2e:go_default_test --test_output=all --test_arg=-test.v
```

[gitops rules]: https://github.com/adobe/rules_gitops#:~:text=Bazel%20GitOps%20Rules,kustomize%20overlays%20for%20their%20services.
[Loki]: https://github.com/grafana/loki
[kind]: https://github.com/kubernetes-sigs/kind
[bazelisk]: https://github.com/bazelbuild/bazelisk

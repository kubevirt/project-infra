# ci-search

Customization and deployment of [Openshift's ci-search] on Kubevirt CI cluster.
It uses internally these bazel [gitops rules].

## Deployment

You need:
* a kubernetes configuration at `~/.kube/config` with an user allowed to
create [these resources](./manifests).
* [bazelisk] installed.

Then, from the root of project-infra run:
```
$ ./github/ci/services/ci-search/hack/deploy.sh production
```

## Tests

Can be tested locally using [kind] and [bazelisk], from the root of project-infra:
```
$ kind create cluster
$ ./github/ci/services/ci-search/hack/deploy.sh testing
$ bazelisk test //github/ci/services/ci-search/e2e:go_default_test --test_output=all --test_arg=-test.v
```

[gitops rules]: https://github.com/adobe/rules_gitops#:~:text=Bazel%20GitOps%20Rules,kustomize%20overlays%20for%20their%20services.
[Openshift's ci-search]: https://github.com/openshift/ci-search
[kind]: https://github.com/kubernetes-sigs/kind
[bazelisk]: https://github.com/bazelbuild/bazelisk

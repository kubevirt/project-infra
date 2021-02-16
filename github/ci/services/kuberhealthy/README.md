# ci-search

Customization and deployment of [Kuberhealthy]. It uses internally
these bazel [gitops rules].

## Deployment

You need:
* a kubernetes configuration at `~/.kube/config` with an user allowed to
create [these resources](./manifests).
* [bazelisk] installed.

Then, from the root of project-infra run:
```
$ ./github/ci/services/kuberhealthy/hack/deploy.sh production
```

## Tests

Can be tested locally using [kind] and [bazelisk], from the root of project-infra:
```
$ kind create cluster
$ ./github/ci/services/kuberhealthy/hack/deploy.sh testing
$ bazelisk test //github/ci/services/kuberhealthy/e2e:go_default_test --test_output=all --test_arg=-test.v
```

[gitops rules]: https://github.com/adobe/rules_gitops#:~:text=Bazel%20GitOps%20Rules,kustomize%20overlays%20for%20their%20services.
[Kuberhealthy]: https://github.com/Comcast/kuberhealthy
[kind]: https://github.com/kubernetes-sigs/kind
[bazelisk]: https://github.com/bazelbuild/bazelisk

# ci-search

Customization and deployment of [Openshift's ci-search] on Kubevirt CI cluster.

## Deployment

You need:
* a kubernetes configuration at `~/.kube/config` with an user allowed to
create [these resources](./base/manifests).
* golang and kubectl installed.

Then, from the root of project-infra run:
```
$ ./github/ci/services/ci-search/hack/deploy.sh production
```

## Tests

Can be tested locally using [kind] and [go], from the root of project-infra:

```bash
$ kind create cluster
$ ./github/ci/services/ci-search/hack/deploy.sh testing
$ go test ./github/ci/services/ci-search/e2e -v
```

[Openshift's ci-search]: https://github.com/openshift/ci-search
[kind]: https://github.com/kubernetes-sigs/kind

kubevirtci-bumper
===============

`kubevirtci-bumper` is a tool to automatically bump kubevirtci providers.

`kubevirtci-bumper` does the following:

1. Ensuring that we have a provider for the latest minor release by copying a predecessor provider
2. Ensuring that the last three minor releases are up-to-date
3. Dropping unsupported k8s providers automatically

Example
=======

```
$ # Ensure that we have the latest provider in kubevirtci
$ kubevirtci-bumper --github-token-path="" --ensure-latest --k8s-provider-dir /go/src/kubevirt.io/kubevirtci/cluster-provision/k8s
  {"level":"info","msg":"Added provider 1.17 with version 1.17.3","time":"2020-02-19T11:46:15+01:00"}
  {"level":"info","msg":"Minor version 1.17 updated to 1.17.3","time":"2020-02-19T11:46:15+01:00"}
$ release-querier --github-token-path="" -org=kubernetes -repo=kubernetes -latest
v1.17.3
$
$ # Ensure that we point to the latest patch release of the last three supported minor k8s releases
$ kubevirtci-bumper --github-token-path="" --ensure-last-three-minor-of v1 --k8s-provider-dir /go/src/kubevirt.io/kubevirtci/cluster-provision/k8s
  {"level":"info","msg":"Minor version 1.17 updated to 1.17.3","time":"2020-02-19T11:47:17+01:00"}
  {"level":"info","msg":"Minor version 1.16 updated to 1.16.7","time":"2020-02-19T11:47:17+01:00"}
  {"level":"info","msg":"Minor version 1.15 updated to 1.15.10","time":"2020-02-19T11:47:17+01:00"}
```

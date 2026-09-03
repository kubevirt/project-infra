# Docker Mirror Proxy

## Motivation

To save bandwidth and optimize performance, the KubeVirt CI infrastructure uses
a caching proxy to download container images.

For this purpose, the open-source project [docker-registry-proxy] is used.

[docker-registry-proxy]: https://github.com/rpardini/docker-registry-proxy

## Deployment

The manifests used to deploy the `docker-mirror-proxy` service are stored in
KubeVirt CI project-infra git repository under the Prow deployment folder:
[github/ci/prow-deploy/kustom/components/docker-mirror-proxy](https://github.com/kubevirt/project-infra/tree/main/github/ci/prow-deploy/kustom/components/docker-mirror-proxy).

It is currently provided on the following KubeVirt CI Prow build clusters:
- kubevirt-prow-control-plane
- prow-workloads

## Usage

Several KubeVirt CI subsystems have been adapted to take advantage of the
`docker-mirror-proxy` service.

### Prow Jobs

To use the `docker-mirror-proxy` service, Prow Jobs can set the
[`preset-docker-mirror-proxy`] preset.

[`preset-docker-mirror-proxy`]: https://github.com/kubevirt/project-infra/blob/4969b3b122efb06f6c3473e41e2a40121b018638/github/ci/prow-deploy/kustom/base/configs/current/config/config.yaml#L747-L772

This defines environment variables and populates configuration files which are
used by the [runner] script of containers images based on KubeVirt CI bootstrap
image (such as the one for Golang) to configure Docker and Podman to pull
container images through the proxy.

[runner]: https://github.com/kubevirt/project-infra/blob/main/images/bootstrap/runner.sh

### k8s cluster-up providers

KubeVirt CI `gocli` supports configuring the `CRI-O` service of the k8s
cluster-up providers to pull container images via a proxy. It can be enabled
using the [`docker-proxy`] command-line flag.

[`docker-proxy`]: https://github.com/kubevirt/kubevirtci/tree/main/cluster-provision/gocli/opts/docker-proxy

### kind cluster-up providers

KubeVirt CI kind cluster-up providers have a [configure-registry-proxy] script
that supports configuring the `containerd` service to pull container images via
a proxy. It is invoked when the `CI` environment variable to `true`.

[configure-registry-proxy]: https://github.com/kubevirt/kubevirtci/blob/main/cluster-up/cluster/kind/configure-registry-proxy.sh

This variable is populated by default in jobs created by Prow.

## Certificate management

Prow jobs will fail to pull container images if the docker-mirror-proxy CA
certificate is allowed to expire.

There is a [periodic job] in place to check that the CA certificate will not
expire within the next 90 days.

[periodic job]: https://github.com/kubevirt/project-infra/blob/4969b3b122efb06f6c3473e41e2a40121b018638/github/ci/prow-deploy/files/jobs/kubevirt/project-infra/project-infra-periodics.yaml#L877

Once this periodic check starts to fail, a maintenance window should be
scheduled to allow for renewing this CA certificate.

There is a script in the docker-mirror-proxy repo that simplifies the process
for the CA cert renewal - [create-ca-cert.sh].

[create-ca-cert.sh]: https://github.com/rpardini/docker-registry-proxy/blob/master/create_ca_cert.sh

## Steps to renew the CA certificate

* Create directory to mount into docker-mirror-proxy container
  ```
  cd /tmp && mkdir ./ca
  ```
* Run the docker-mirror-proxy container locally
  ```
  podman container run --privileged -v /tmp/ca:/ca:rw ghcr.io/rpardini/docker-registry-proxy:0.7.0
  ```
* The `ca.crt` and `ca.pem` should be created under `/tmp/ca/`
* Add these files to the secrets repository
* Once they have been merged to the secrets repo, trigger the prow-deploy
  postsubmit jobs to apply the updated secrets
* Restart the docker-mirror-proxy pod in the cluster to ensure that it has the
  updated CA certificate

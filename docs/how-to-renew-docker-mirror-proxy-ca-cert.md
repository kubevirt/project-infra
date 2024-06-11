# Renewing the CA Certificate for the docker-mirror-proxy

Prow jobs will fail to pull container images if the docker-mirror-proxy CA certificate is allowed to expire. 

There is a [periodic job](https://github.com/kubevirt/project-infra/blob/dcd4fd00a2c1592e0db577bd824314cdd4221818/github/ci/prow-deploy/files/jobs/kubevirt/project-infra/project-infra-periodics.yaml#L541) in place to check that the CA certificate will not expire within the next 90 days. 

Once this periodic check starts to fail, a maintenance window should be scheduled to allow for renewing this CA certificate. 

There is a script in the docker-mirror-proxy repo that simplifies the process for the CA cert renewal - [create-ca-cert.sh](https://github.com/rpardini/docker-registry-proxy/blob/master/create_ca_cert.sh)

## Steps to renew the CA certificate

* Create directory to mount into docker-mirror-proxy container
  ```
  cd /tmp && mkdir ./ca
  ```
* Run the docker-mirror-proxy container locally
  ```
  podman run --privileged -v /tmp/ca:/ca:rw ghcr.io/rpardini/docker-registry-proxy:0.6.2
  ```
* The `ca.crt` and `ca.pem` should be created under `/tmp/ca/`
* Add these files to the secrets repository
* Once they have been merged to the secrets repo, trigger the prow-deploy postsubmit jobs to apply the updated secrets.
* Restart the docker-mirror-proxy pod in the cluster to ensure that it has the updated CA certificate

# How we execute CI jobs on different clusters

Our Prow jobs can be configured to run on different kubernetes clusters using 2
methods:

* Using the build clusters defined in Prow components config: explained in
[this Prow document]. Basically Prow uses a kubeconfig with a context for each
of the build clusters, keep in mind that the user must be admin of the namespace
where Prow is configured to run jobs. Prow components (at minimum deck, sinker
and prow-controller-manager) mount this kubeconfig secret to have access to the
different clusters.

*IMPORTANT*: given that some Prow components can misbehave if any of the build
clusters is not responding, for an external cluster to be considered we require
it to have an HA control plane with at least 3 nodes.

* For jobs that rely on KubeVirtCI, use the external provider feature: KubeVirtCi
allows to spin up a remote test cluster using the environment variables
`KUBEVIRT_PROVIDER` set to `external` and `KUBECONFIG` set to the path of a
kubeconfig file with admin access to the remote cluster.

# How to add new clusters to run CI jobs

For each kind of external clusters the process is different.

## Prow federated build cluster

### Cluster administrator: provide a `kubeconfig`

Obtain a `kubeconfig` for your cluster that refers to a user with admin permissions on the namespace where Prow is configured to create jobs (currently `kubevirt-prow-jobs`).

If you are not a Kubevirt CI maintainer you need to create an
[issue on project-infra] with title `Add Prow build cluster <name>`, being
`<name>` the name you want to use to schedule jobs in the new cluster, and
with the [kubeconfig encrypted as described here] attached to the issue.

### CI maintainer: integrate kubeconfig into prow main kubeconfig entry
* Decrypt the `kubeconfig` file attached to the issue
* Use it to check whether you can connect to the cluster
* Check whether there's a service account created that you can use for the users section inside the kubeconfig
* If there's no service account present, generate it like this (replace names accordingly:
```bash
# create a serviceaccount for prow
kubectl create serviceaccount prow-workloads-cluster-automation
# optional: make serviceaccount cluster-admin
kubectl create clusterrolebinding prow-workloads-cluster-automation \
    --clusterrole=cluster-admin \
    --serviceaccount=default:prow-workloads-cluster-automation
# create token for serviceaccount to use as a user token later
kubectl create -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: prow-workloads-cluster-automation
  namespace: default
  annotations:
    kubernetes.io/service-account.name: prow-workloads-cluster-automation
type: kubernetes.io/service-account-token
# export secret token data to add to user later on
kubectl get secret prow-workloads-cluster-automation \
    -n "default" -o yaml | \
    yq -r '.data.token' | \
    base64 -d
EOF
```
> [!WARNING]
> It is advised to reduce the access for the service account to admin permissions for the namespace where Prow needs to create jobs (see [above](#cluster-administrator-provide-a-kubeconfig))

* For a new cluster generate new sections inside `kubeconfig` for the new cluster:
```yaml
apiVersion: v1
clusters:
...
- cluster:
    certificate-authority-data: {kubeconfig-cert-auth-data}
    server: {kubeconfig-server}
  name: {cluster-name}
...
contexts:
...
- context:
    cluster: {cluster-name}
    user: {serviceaccount-name}
  name: {context-name}
users:
...
- name: {serviceaccount-name}
  user:
    token: {serviceaccount-secret-data}
```
* Decrypt the secrets in https://github.com/kubevirt/secrets and add the
  user and cluster in the new kubeconfig to a new context named `<name>` in
  the `kubeconfig` entry of the secrets.
* Encrypt the secrets and create a PR to https://github.com/kubevirt/secrets
   with the changed file.
* Once the PR is merged write a comment on the original issue saying that
all is done and close it.

Now we can create Prow jobs that have the `cluster` field set to the name we provided above.

## KubevirtCI external provider

  * Obtain a kubeconfig for your cluster that refers to an user with admin
  permissions.

  * Include your kubeconfig in our automation secrets:

    * If you are not a Kubevirt CI maintainer you need to create an
    [issue on project-infra] with title `Add KubeVirtCI external provider <name>`,
    being `<name>` the name you want to use to identify your cluster, and with the
    [kubeconfig encrypted as described here] attached to the issue.
    * If you are a CI maintainer you should:

      * Decrypt the kubeconfig file attached to the issue.
      * Decrypt the secrets in https://github.com/kubevirt/secrets and add the
      new kubeconfig in a new entry of the secrets named `<name>`.
      * Encrypt the secrets and create a PR to https://github.com/kubevirt/secrets
      with the changed file.
      * Once the PR is merged write a comment on the original issue saying that
      all is done and close it.

  * Create Prow jobs that extract the cluster kubeconfig from the automation
  secrets and defines as environment variables `KUBEVIRT_PROVIDER` set to `external`
  and `KUBECONFIG` set to the path of the extracted  kubeconfig. Here you can
  check [an example of a job configured with KubeVirtCI external provider].

# <a name="encrypt"></a>How to encrypt the kubeconfig of your cluster

First you need to import the public part of the key we use on our automation:

```bash
gpg --recv-keys 99F7C0D2E1BB8025
```
Then, assuming you saved your kubeconfig in `/path/to/my-kubeconfig.yaml`, you can
encrypt it for our automation key with:
```bash
gpg --output my-kubeconfig.gpg --encrypt --recipient 99F7C0D2E1BB8025 /path/to/my-kubeconfig.yaml
```
The encrypted file will be available in `my-kubeconfig.gpg` in the current
directory.


[this Prow document]: https://github.com/kubernetes/test-infra/blob/master/prow/getting_started_deploy.md#Run-test-pods-in-different-clusters
[issue on project-infra]: https://github.com/kubevirt/project-infra/issues/new
[kubeconfig encrypted as described here]: #encrypt
[an example of a job configured with KubeVirtCI external provider]: https://github.com/kubevirt/project-infra/blob/main/github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml#L1206

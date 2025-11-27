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

### Cluster administrator

#### Prepare cluster for Prow

There's three basic prerequisites for adding a cluster to our federation:
1. the namespace `kubevirt-prow-jobs` needs to exist
2. a service account (suggested name `prow-workloads-cluster-automation`) needs to exist and have a secret attached
3. the service account needs to be able to manage pods and secrets in namespace `kubevirt-prow-jobs`

#### Create namespace for kubevirt prow

```bash
kubectl create namespace kubevirt-prow-jobs
```

#### Create service account for kubevirt prow

Generate it like this (replace names accordingly if required):

```bash
# create a serviceaccount for prow
kubectl create serviceaccount -n kubevirt-prow-jobs prow-workloads-cluster-automation
# create token for serviceaccount to use as a user token later
kubectl create -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: prow-workloads-cluster-automation
  namespace: kubevirt-prow-jobs
  annotations:
    kubernetes.io/service-account.name: prow-workloads-cluster-automation
type: kubernetes.io/service-account-token
EOF
# export secret token data to add to user later on
kubectl get secret prow-workloads-cluster-automation \
    -n "kubevirt-prow-jobs" -o yaml | \
    yq -r '.data.token' | \
    base64 -d
```

#### Give access to service account

> [!WARNING]
> It is advised to reduce the access for the service account to minimum permissions for the namespace where Prow needs to create jobs.
> Making the serviceaccount admin is a compromise, so that we can create secrets and other changes if required

```bash
# make serviceaccount admin on namespace
kubectl create rolebinding kubevirt-prow-workloads-admin \
    --namespace kubevirt-prow-jobs \
    --clusterrole=admin \
    --serviceaccount=kubevirt-prow-jobs:prow-workloads-cluster-automation
```

#### Provide a `kubeconfig`

Create a `kubeconfig` for your cluster that uses the created service account as user.

If you are not a Kubevirt CI maintainer you need to create an
[issue on project-infra] with title `Add Prow build cluster <name>`, being
`<name>` the name you want to use to schedule jobs in the new cluster, and
with the [kubeconfig encrypted as described here] attached to the issue.

### CI maintainer

#### Integrate kubeconfig into prow main kubeconfig entry
* Decrypt the `kubeconfig` file attached to the issue
* Check whether you can connect to the cluster with it
* Check whether namespace and service account are created and can manage pods and secrets, so that it can be used inside the users section of the kubeconfig
* Decrypt the secrets in https://github.com/kubevirt/secrets and add the
  user and cluster in the new kubeconfig to a new context named `<name>` in
  the `kubeconfig` entry of the secrets.

  For a new cluster generate new sections inside `kubeconfig` for the new cluster:
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
* Encrypt the secrets and create a PR to https://github.com/kubevirt/secrets
   with the changed file.
* Once the PR is merged write a comment on the original issue saying that
all is done and close it.

Create secrets that are required for the jobs.

> [!NOTE]
> Currently there's only the `gcs` secret required.

Now we can create Prow job configurations that have the `cluster` field set to the name we provided above.

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


[this Prow document]: https://github.com/kubernetes-sigs/prow/blob/main/site/content/en/docs/getting-started-deploy.md#run-test-pods-in-different-clusters
[issue on project-infra]: https://github.com/kubevirt/project-infra/issues/new
[kubeconfig encrypted as described here]: #encrypt
[an example of a job configured with KubeVirtCI external provider]: https://github.com/kubevirt/project-infra/blob/bab947fa42f89f78238160d487bf047f4dea5c9f/github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml#L650

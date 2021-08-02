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

# How can we add a new cluster to run CI jobs

For each kind of external clusters the process is different:

* Prow build clusters:

  * Obtain a kubeconfig for your cluster that refers to an user with admin
  permissions on the namespace where Prow is configured to create jobs (currently
  `kubevirt-prow-jobs`).

  * Update Prow's kubeconfig secret:

    * If you are not a Kubevirt CI maintainer you need to create an
    [issue on project-infra] with title `Add Prow build cluster <name>`, being
    `<name>` the name you want to use to schedule jobs in the new cluster, and
    with the [kubeconfig encrypted as described here] attached to the issue.
    * If you are a CI maintainer you should:

      * Decrypt the kubeconfig file attached to the issue.
      * Decrypt the secrets in https://github.com/kubevirt/secrets and add the
      user and cluster in the new kubeconfig to a new context named `<name>` in
      the `kubeconfig` entry of the secrets.
      * Encrypt the secrets and create a PR to https://github.com/kubevirt/secrets
      with the changed file.
      * Once the PR is merged write a comment on the original issue saying that
      all is done and close it.

  * Create Prow jobs that have the `cluster` field set to the name you provided in
  the issue above.

* KubevirtCI external provider:

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

```
$ gpg --recv-key 0xF0C379576ACC7D14
```
Then, assuming you saved your kubeconfig in `/path/to/my-kubeconfig.yaml`, you can
encrypt it for our automation key with:
```
$ gpg --output my-kubeconfig.gpg --encrypt --recipient 0xF0C379576ACC7D14 /path/to/my-kubeconfig.yaml
```
The encrypted file will be available in `my-kubeconfig.gpg` in the current
directory.


[this Prow document]: https://github.com/kubernetes/test-infra/blob/master/prow/getting_started_deploy.md#Run-test-pods-in-different-clusters
[issue on project-infra]: https://github.com/kubevirt/project-infra/issues/new
[kubeconfig encrypted as described here]: #encrypt
[an example of a job configured with KubeVirtCI external provider]: https://github.com/kubevirt/project-infra/blob/main/github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml#L1206

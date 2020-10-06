# KubeVirt-CI

Ansible based description of the KubeVirt CI environment for functional tests.
The Ansible roles here allow to (re)crate our Prow setup.

## Prepare your Github Project for Prow

1. create an access token with the following permissions, to give prow access
   to your github organization:

![test](personal_access_token.png)

1. create an access token or use the one you created above. Prow needs the
   `public_repo` permission.
2. register the Prow callback URL in your github project

 * Fill in your callback URL (e.g. `https://prow.myopenshift.com/hook/`)
 * Content-type should be `application/json`
 * Create a secret with `openssl rand -hex 20` and set it on the webhook
 * Enable all notifications

Note that the included route for prow hooks will be exposed via `https`.

### Prepare your Ansible Variables

Create a file `group_vars/all/main.yml` based on

```yaml
---
remoteClusterName: ""
remoteClusterProwJobsContext: ""
masterClusterContext: ""
kubeconfig: |
  # kubeconfig to be used by automation accounts
remoteClusterEndpoint: ""
githubSecret: ""
githubToken: "453f86e8a6c9eed45789c689089e1eb2w9x2fda3"
prowUrl: "deck-prow.e8ca.engint.openshiftapps.com" # without the /hook subpath
prowNamespace: "prow"
prowHmac: "e4a61a12b5cae91dca3b8c1a576c735fe971110f" # the webhook secret generated
prowAdmins: [ "username" ]
# create a github application in your org and fill in the secrets
appOAuthConfig: |
  client_id: myid
  client_secret: mysecret
  redirect_url: https://deck-url/github-login/redirect
  final_redirect_url: https://deck-url/pr
# openssl rand -base64 64
appCookieSecret: |
  6G38MhKmwr6AR9je1YbZfiSSHEMazBzItMHfA0XeiQNzNaSw/ACp05WqAIOUQR60
XlA8HciTwAh/+pNhR7aquA==
# gce service-account.json with bucket admin rights
# this is the usual service account format when creating a gce service account
gcsServiceAccount: |
  {
    "type": "service_account",
    "project_id": "project-id",
    "private_key_id": "private-key-id",
    "private_key": "private-key",
    "client_email": "email",
    "client_id": "client-id",
    "auth_uri": "https://accounts.google.com/o/oauth2/auth",
    "token_uri": "https://oauth2.googleapis.com/token",
    "auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
    "client_x509_cert_url": "cert-url"
  }
# the OKD installer pull token secret
installerPullToken: 'pullSecret: {"auths":{"cloud.openshift.com":{"auth":"test","email":"test@test.com"},"quay.io":{"auth":"test","email":"test@test.com"}}}'
```

## Run the Playbook

Add your master and your clients to the `inventory` file:

```
[local]
localhost ansible_connection=local
```

`[prow]` will use your local kubectl and oc binaries to deploy the master cluster.

`[remote-cluster]` will use your local kubectl and oc binaries to deploy the remote cluster.

The credentials should be provided in your group_vars kubeconfig section.

<b>Note about the order of deployment</b>

The two playbooks are independant. You can run either one of them when you need.

### Configure the remote cluster

```
ansible-playbook -i inventory remote-cluster.yml
```

### Configure the master cluster


```
ansible-playbook -i inventory prow.yaml
```

### How to share secrets?

The project's secrets are stored in an encrypted format in a private GitHub repo.
If you need access to the repo, please reach to one of the project's maintainers
or send an email to our mailing list with an explanation why you need access to
the project's secrets. 

#### Error: unable to load Private Key
If you are getting an error of the following:

```bash
❯ ./decrypt.sh ~/.ssh/id_rsa dhiller
unable to load Private Key
139888015738688:error:0909006C:PEM routines:get_name:no start line:crypto/pem/pem_lib.c:745:Expecting: ANY PRIVATE KEY
```

the reason is likely that your key is not in PEM format. According to the documentation keys created by `ssh-keygen` are created in OpenSSH format by default (i.e. file starting with `-----BEGIN OPENSSH PRIVATE KEY-----`).

You can convert the OpenSSH key into PEM format by doing the following:

```bash
# make a copy of the key as the conversion will be done inline
cp ~/.ssh/id_rsa /tmp/

# convert it to pem format inline (asking for old password and new password)
ssh-keygen -p -f /tmp/id_rsa -m PEM
mv /tmp/id_rsa ~/.ssh/id_rsa.pem
```
The resulting file should look like this:

```bash
-----BEGIN RSA PRIVATE KEY-----
...
```

Then you can use the decrypt script again like above, except that you use the pem key:

```bash
> ./decrypt.sh ~/.ssh/id_rsa.pem dhiller
```

### Testing new ProwJobs

The tool `mkpj` can be used to create jobs out of local configurations. Then
simply post them to the cluster:

```
go get k8s.io/test-infra/prow/cmd/mkpj
mkpj --pull-number <pr-number> -job <job-name>  -job-config-path github/ci/prow/files/jobs/ --config-path github/ci/prow/files/config.yaml > job.yaml
oc create -f job.yaml -n kubevirt-prow-jobs
```

### Forking kubevirt/kubevirt jobs on release

Run

```
git clone git@github.com:kubernetes/test-infra.git
cd test-infra
bazel run //experiment/config-forker -- --job-config ~/project-infra/jobs/kubevirt/kubevirt-presubmits.yaml   --version ${RELEASE_VERSION} --output ~/project-infra/jobs/kubevirt/kubevirt-presubmits-${RELEASE_VERSION}.yaml
```

For more details see https://github.com/kubernetes/test-infra/blob/master/experiment/config-forker/README.md


### Updating prow configuration manually

In case the config-updater doesn't run after merging a PR into 
project-infra (symptom is i.e the comment from kubevirt-bot that it updated the configuration is missing), we need to update the complete prow configuration 
manually.

To manually update, go to your [kubernetes/test-infra](https://github.com/kubernetes/test-infra/) directory and execute:

```bash
( CONFIG_DIR=$(cd path-to/project-infra && pwd); \
    bazel run //prow/cmd/config-bootstrapper -- \
        --dry-run=false \
        --source-path=$CONFIG_DIR  \
        --config-path=$CONFIG_DIR/github/ci/prow/files/config.yaml \
        --plugin-config=$CONFIG_DIR/github/ci/prow/files/plugins.yaml \
        --job-config-path=$CONFIG_DIR/github/ci/prow/files/jobs \
        --deck-url=https://prow.apps.ovirt.org )
```

See [config-bootstrapper](https://github.com/kubernetes/test-infra/tree/master/prow/cmd/config-bootstrapper)


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
```

## Run the Playbook

Add your master and your clients to the `inventory` file:

```
[prow]
localhost ansible_connection=local
```

`[prow]` will use your local openshift
credentials and deploy prow on the configured cluster.

Provision prow:

```
ansible-playbook -i inventory prow.yaml
```
### How to share secrets?

There is an `encrypt.sh` script include. For instance to share the needed
secrets with the user `nobody`, us the public key of the user and run:

```bash
$ ./encrypt.sh ~/.ssh/nobody.pub nobody
Adding the following files:

  secrets/
  secrets/group_vars/
  secrets/group_vars/all/
  secrets/group_vars/all/main.yml
  secrets/inventory

Created nobody.tar.enc and nobody.key.enc
```

The receiver can decrypt the aes key via her public key and then use the aes
key to decrypt the tar file:

```bash
./decrypt.sh ~/.ssh/id_rsa rmohr
File rmohr.tar decrypted.
```

### Testing new ProwJobs

The tool `mkpj` can be used to create jobs out of local configurations. Then
simply post them to the cluster:

```
go get k8s.io/test-infra/prow/cmd/mkpj
mkpj --pull-number <pr-number> -job <job-name>  -job-config-path github/ci/prow/files/jobs/ --config-path github/ci/prow/files/config.yaml > job.yaml
oc create -f job.yaml -n kubevirt-prow-jobs
```

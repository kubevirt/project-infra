# KubeVirt-CI

Ansible based description of the KubeVirt CI environment for functional tests.
The Ansible roles here allow to re-crate and scale the Jenkins CI environment
used.

## Prepare your Github Project

The following steps **1** and **2** give your Jenkins server the permissions to
add comments to your Pull Request and to change the build status. Step **3**
allow your Jenkins server, to be notified by Github, if there is new code to
test. Independently of these steps, periodically polling a Github repository
for changes will always work.

1. create an access token with the following permissions, to allow your Jenkins
   server to update the PR states.

![test](personal_access_token.png)

2. add your Bot to your github project.
3. register the Jenkins callback URL in your github project

 * Fill in your callback URL (e.g. `http://myjenkins.com:1234/ghprbhook/`)
 * Content-type should be `application/x-www-form-urlencoded`
 * Add a secret
 * Enable `Push`, `Pull request`, `Issue comment` notifications

## Prepare your Ansible Variables

Create a file `group_vars/all/main.yml` based on

```
---
jenkinsUser: "jenkins"
jenkinsPass: "mypwd"
master: "http://my.jenkins.com:8080"
slaveSlots: 1
githubSecret: ""
githubCallbackUrl: "http://my.jenkins.com:8080"
githubToken: "453f86e8a6c9eed45789c689089e1eb2w9x2fda3"
githubRepo: "rmohr/kubevirt"
```

There you can fill in you token, your secret and the Jenkins callback URL.

## Scaling

To add new workers, a client role exists. It uses the Jenkins Swarm plugin to
attach it to the Jenkins master.

## Run the Playbook

Add your master and your clients to the `hosts` file:

```
[jenkins-master]
master ansible_host=my.jenkins.com ansible_user=root

[jenkins-slaves]
slave0 ansible_host=slave0.my.jenkins.com ansible_user=root
slave1 ansible_host=slave1.my.jenkins.com ansible_user=root
```

Provision your machines:

```
ansible-playbook -i hosts ci.yaml
```

## KubeVirt CI Landscape Specifics

There exists an additional `beaker` role. It is not generalized, and allows us
in all our beaker managed servers, to increase the LVM volumes to the maximum
available size. The resulting extra LVM volume, is then used as the default
storage location for all VM images.

To make use of that role, adjust your `hosts` file and add all beaker managed
servers to a `beaker` section:

```
[beaker]
slave0 ansible_host=slave0.my.jenkins.com ansible_user=root
slave1 ansible_host=slave1.my.jenkins.com ansible_user=root
```

## Testing the CI infrastructure

To test changes in this setup, an extra Vagrant setup exists in this repository. To
provision a master and a slave with Vagrant, do the following:

```bash
mkdir -p group_vars/all/
cat >group_vars/all/main.yml <<EOL
jenkinsUser: "vagrant"
jenkinsPass: "vagrant"
master: "http://192.168.201.2:8080"
slaveSlots: 1
githubSecret: ""
githubCallbackUrl: "http://my.jenkins.com:8080"
githubToken: "453f86e8a6c9eed45789c689089e1eb2w9x2fda3"
githubRepo: "kubevirt/kubevirt"
EOL
vagrant up
```

This will provision a master and a slave. The master Jenkins instance can be
reached at `http://192.168.201.2:8080` after the Ansible Playbooks are done.
Credentials are `vagrant:vagrant` To re-run the Ansible Playbooks after a
change, it is sufficient to just run `vagrant provision`. So no need to destroy
the whole machines, if you don't want a full test run.

The provisioned VMs, will periodically poll the `kubevirt/kubevirt` repository
and try to run the functional tests. Since the token is not valid, it will not
update the KubeVirt repository. Nested virtualization can be slow, don't expect
too much from it.

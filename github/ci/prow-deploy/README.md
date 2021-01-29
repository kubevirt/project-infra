# Kubevirt Prow deployment

Prow is normally deployed from test-infra repository against the
kubernetes infrastructure
In kubevirt case, we are using some manifests from the prow deployment
and some manifests to deploy prow in three different environment

- kubevirtci-testing

Is the testing environment to test prow both locally or on automatic jobs.
It uses a cluster created by kubvirtci as target to deploy prow

- ibmcloud-staging

Is the staging environment to test prow alongside production.
It uses the cluster in ibmcloud, but with changes in naming and paths to not
interfere with production deployment

- ibmcloud-production

Is the production environment. This is the default environment for production
deployment. A deployment on this environment will affect real jobs

## How to launch a deployment

To launch deployment three tools are needed

- kustomize

Generates manifests from a base, applying a set of patches

- yq

Command line tool to handle yaml files

- kubectl

### Install dependencies

Version 3 of kustomize is needed.

    GOBIN=$(pwd)/ GO111MODULE=on go get sigs.k8s.io/kustomize/kustomize/v3

Or you can follow instructions at https://kubectl.docs.kubernetes.io/installation/kustomize/source/

The install yq. Please be aware that there is another yq written in python
with the same goal, but it doesn't support patch scripting and it's not usable here.

    GO111MODULE=on go get github.com/mikefarah/yq/v3

yq (the one in go, not the one in python)

### Generate configuration

Before being able to generate the ConfigMaps with kustomize, we need to generate
environment specific configuration from the base configuration. This is done
separately with yq as kustomize deals with kubernetes manifests only, not generic yaml files.

There's a script in the base kustom directory to automate this.

    kustom/render-environment-configs.sh <environment>

Will apply patches to the configuration as defined in the patch script for the environment in

    kustom/overlays/$environment/yq_scripts

then renders the yaml configurations, then copy them to the environment directory at

    kustom/overlays/$environment/configs

### Copy the secrets

Before being able to generate the secrets with kustomize, we need to copy them in the
proper overlay directory at

    kustom/overlays/$overlay/secrets

All the secrets needed are contained under the directory

    kustom/secrets-boilerplate

Copy the whole subtree to the environment dir then fill out all your secret.
No code should be pushed as PR which contains environment specific secrets.
As an additional security measure, the environment specific secret directories are
ignored explicity with a .gitignore.

### Generate manifests

When configuration and secrets are in place in the overlay specific directories,
we can finally call kustomize to generate overlay specific manifests:

    ~/go/bin/kustomize build kustom/overlays/$overlay > prow-deploy.yaml

WARNING: There is a version of kustomize that is embedded in kubectl, but it's not the version
required by this deployment, so don't use it.

After the manifests are rendered without errors, you can directly apply them with kubectl

    kubectl apply -f prow-deploy.yaml

## Kustomize structure

The kustomize structure is contained under the kustom directory inside prow-deploy role.

- base

Contains the base configurations and manifests

- base/kustomization.yaml

Contains the list of resources utilized in the prow deployment. Only manifests
specified here will be included in the final kustomized rendering.

- base/manifests/test-infra

Contains an exact copy of the prow manifests from the test-infra repository.
They will be in a directory named as the git SHA from where they were pulled from

- base/manifests/local

Will contain the manifests created specifically for the kubevirt prow deployment.

- base/config

Will contain the base yaml configuration files for the deployments. They are under
a timestamped directory to make config versioning easier.

- overlays

Will contain the overlay specific configurations and patches.

- overlays/$overlay/configs

Will contain the rendered configurations

- overlays/$overlay/secrets

Will contain the copied secrets.

- overlays/$overlay/yq_scripts

Contains patch scripts for the yq tool, to modify base configuration files

- overlays/$overlay/patches

Contains the patches to modify base manifests, divided per patch type.

### Kustomize patches

The main target of kustomization are names and namespaces, and generating
resources to not conflict with production.
So the patches are focusing on namespaces and changing paths.
The option "namespace" offered by customize cannot be used here, as it works
well only when there's a single namespace to be considered, and overrides ALL
namespaces present.
The option "prefix" is used in staging overlay, but it's not enough as some
resources configuration retain the old names.


## Testing

# Prow deployment role

This role deploys prow in a kubernetes cluster created using kubevirt-ci
It contains variables that define a set of manifests to upload
to deploy prow, then uses kubernetes_crud role as primary interface with
the staging cluster to upload the manifests.

## Role structure

    molecule/default

Contains the main scenario for local or automated testing.

Beside invoking kustomize, the role performs small setup/cleanup
tasks, primarily passing variables around.

## How to test the role

The role is tested using molecule.
Molecule will take care of all the test task.
As a prerrequisite you need two environment variables defined:
* `GITHUB_TOKEN`: should contain the path of a valid github account token, any token would do,
no need to have any specific permissions.
* `GOOGLE_APPLICATION_CREDENTIALS` with the path of a Google Cloud Platform JSON credentials file; as with
the github token there are no specific requirements in terms of permissions.

With these environment variables exported you need to create a virtual environment, from `github/ci/prow-deploy` run:

    $ python3 -m venv venv

Now you can activate the virtual environment and install the dependencies:

    $ source ./venv/bin/activate
    $ pip install -r requirements.txt

Then you can run:

    $ molecule prepare

To launch the kubevirtci cluster and prepare the nodes
properly. This is a protected action, it cannot be done twice.
The natural flow is that you can prepare again an instance only
after you destroy it, so if you need to prepare again but no destroy
has been issued, you need to call

    $ molecule reset

To tell molecule to start from scratch.

Then start deployment with

    $ molecule converge

This will launch the prow deployment itself, will wait for the deployment
to settle and then will collect some information in the
artifacts dir.

    $ molecule verify

Will launch a set of tests to verify that the deployment
works correctly. At the moment only smoke tests are available.

You can enter the test instance and access the deployed cluster with:

    $ molecule login
    # export KUBECONFIG=/workspace/repos/project-infra-master/github/ci/prow-deploy/kustom/overlays/kubevirtci-testing/secrets/kubeconfig

then you can execute kubectl commands as usual:

    # kubectl get pods --all-namespaces

Additional molecule commands:

    molecule cleanup

will remove prow-namespace, so that prow can be eventually
deployed again in the same cluster

    molecule destroy

will tear down the kubevirt ci cluster completely

    molecule test

will launch all the above step automatically in sequence.

## How to debug the services in live cluster

Behaviour of prow services in pods proved to be less than reliable, with
services unable to start properly, but not reporting any errors in the logs
and marking the pod as "Running"

In those cases, the best way to debug a service is to jump directly on the pod
executing a shell and running the service manually.

### Process overview

We need to enter the pod, download the service code, and run it manually with "go run".
The service will log to stdout and any error will block the execution, giving reason
on its behaviour
Unfortunately, there is no way to stop the pod entrypoint, as any attempt to interfere with
process 1 (the service) will cause the pod termination, but it's possible to start
another process with the same service in parallel.

### Container base image caveat

The container base image of a service is a quite obsolete version of alpine (3.6).
The base image uses musl libc instead of glibc and has very old version of basic tools.
The pod entrypoint is normally a statically compiled binary that requires limited memory and doesn't
rely on the (very old) dynamic libraries offered by the base images when launched.
We need to launch the service using "go run" instead of bazel, because bazel requires glibc.

### Increase pod memory

Launching service with "go run" will require more memory that the default 512Mb
So we need to change the request and limit on the deployment configuration of the pod
you want to debug to at least 1G. If a "go run" attempt is killed by an unexplained signal, more
memory will be needed.
This can be done on the manifests, or directly editing the deployment.
If the manifest is changed the deployment will need to be cleaned up and restart.
If it is done directly, kubernetes will redeploy the pods automatically, but a cleanup will
wipe the configuration.

### Update base image

Once memory requirements are satisfied, the next step is to git clone the service code and install
go runtime.
The go runtime provided with the base alpine image is very old (pre 1.11) and will not be able to run
the code correctly, so the release must be updated.
A script at

    files/debug_prepare.sh

will do this automatically
it is enough to launch it inside the pod, as in the following example

    cd $KUBEVIRTCI_DIR
    POD=deck-74fd5d678d-g8dz6
    ./cluster-up/kubectl.sh -n kubevirt-prow cp debug_prepare.sh $POD:/
    ./cluster-up/kubectl.sh -n kubevirt-prow exec $POD -- chmod +x /debug_prepare.sh
    ./cluster-up/kubectl.sh -n kubevirt-prow exec $POD -- sh -c /debug_prepare.sh

The script will replace repository to a viable version, update the packaging tool,
install basic tools, and clone the test-infra code inside the container.
Once the script has finished, we can run a interactive shell, change to the parent directory of
the service, then launch the service

    ./cluster-up/kubectl.sh -n kubevirt-prow exec -it $POD -- sh
    # cd /srv/test-infra/prow/cmd
    # go run ./$COMMAND

Logs will show on stdout.

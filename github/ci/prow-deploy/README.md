# Kubevirt Prow deployment

This directory contains code to test and deploy Prow and related components
in our clusters.

## Continuous Delivery of Prow

There are three Prow jobs defined in `kubevirt/project-infra` that help us to
implement Continuous Delivery of the components we use in our Prow setup:

* `pull-project-infra-prow-deploy-test`: presubmit that deploys Prow on a test
environment with the same code used for production and executes integration tests.
This job is triggered when files under `github/ci/prow-deploy` directory are
changed on a PR.

* `post-project-infra-prow-control-plane-deployment`: postsubmit that deploys
Prow and related components when changes to files under `github/ci/prow-deploy`
are merged.

* `periodic-project-infra-prow-bump`: periodic that gets the latest Prow manifests
from `kubernetes/test-infra`, bumps the tags of the Prow images used in our
setup and proposes a PR with the changes.

Besides additional changes proposed by collaborators, the normal workflow for
updating Prow consists of `periodic-project-infra-prow-bump` proposing a PR with
changes to update to the latest Prow version. This triggers `pull-project-infra-prow-deploy-test`
and the integration tests are executed. After a review by collaborators the
changes are merged, which triggers `post-project-infra-prow-control-plane-deployment`
to deploy them to production.

## How deployment works

To launch deployment three tools are needed

- kustomize

Generates manifests from a base, applying a set of patches

- yq

Command line tool to handle yaml files

- kubectl

### Configuration generation

Before being able to generate the ConfigMaps with kustomize, we need to generate
environment specific configuration from the base configuration. This is done
separately with yq as kustomize deals with kubernetes manifests only, not generic yaml files.

There's a script in the base kustom directory to automate this.

    kustom/render-environment-configs.sh <environment>

Will apply patches to the configuration as defined in the patch script for the environment in

    kustom/overlays/$environment/yq_scripts

then renders the yaml configurations, then copy them to the environment directory at

    kustom/overlays/$environment/configs

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

- base/manifests/test_infra/current

Contains an exact copy of the prow manifests from the test-infra repository.

- base/manifests/local

Will contain the manifests created specifically for the kubevirt prow deployment.

- base/config

Will contain the base yaml configuration files for the deployments.

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

- components

Kustomize components reusable in the different overlays.

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
    # export KUBECONFIG=/workspace/repos/project-infra-main/github/ci/prow-deploy/kustom/overlays/kubevirtci-testing/secrets/kubeconfig

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

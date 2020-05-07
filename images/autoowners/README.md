autoowners
==========

## What is this?

Tool for automating sync of OWNERS and OWNERS_ALIASES from source repositories to target repository structure.

I.e. for a the github repository `kubevirt/kubevirt` the folder [github/ci/prow/files/jobs/kubevirt/kubevirt](../../github/ci/prow/files/jobs/kubevirt/kubevirt) in this repository contains all job definitions that the prow instance executes. `autoowners` takes care of the OWNERS and OWNERS_ALIASES in said fodler are in sync with what exists in `kubevirt/kubevirt`.

Tool originates from [openshift/ci-tools](https://github.com/openshift/ci-tools). Built manually here as a quick hack to get it in place. Need to create a proper image generation.

## Prerequisites

`DOCKER_PASSWORD` needs to point to file containing the password for the docker image registry,
`DOCKER_USER` needs to point to file containing the user name for the docker image registry

## How to build

Use [build_and_push_from_local.sh](build_and_push_from_local.sh) to build the tool from source, create the image and push it into our kubevirtci docker repository.

Output will be sth like this:

```bash
Pushed autoowners as image kubevirtci/autoowners with digest sha256:025f8ba96ffdc6d3adf17a0058898e17a8fe814314ec3c4bd2af9812aeeda7b7
```

This should then be used as job image for the job `periodic-project-infra-autoowners` [here](../../github/ci/prow/files/jobs/kubevirt/project-infra/project-infra-periodics.yaml).

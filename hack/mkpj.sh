#!/bin/bash

#
# Adapted from https://github.com/kubernetes/test-infra/blob/master/config/mkpj.sh
#
# Usage: hack/mkpj.sh --job <job-name> | kubectl create -n kubevirt-prow-jobs -f -
#

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
config="${root}/github/ci/prow-deploy/files/config.yaml"
job_config_path="${root}/github/ci/prow-deploy/files/jobs"

docker pull gcr.io/k8s-prow/mkpj 1>&2 || true
docker run -i --rm -v "${root}:${root}:z" gcr.io/k8s-prow/mkpj "--config-path=${config}" "--job-config-path=${job_config_path}" "$@"

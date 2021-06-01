#!/bin/bash

set -euo pipefail

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_INFRA_ROOT=$(readlink -f --canonicalize ${BASEDIR}/../../../..)
if [ -d ${PROJECT_INFRA_ROOT}/../../kubernetes/test-infra ]; then
    TEST_INFRA_ROOT=$(readlink -f --canonicalize ${PROJECT_INFRA_ROOT}/../../kubernetes/test-infra)
fi
TESTGRID_GEN_CONFIG=$(readlink -f --canonicalize ${BASEDIR}/../gen-config.yaml)
TESTGRID_CONFIG=$(readlink -f --canonicalize ${BASEDIR}/../config.yaml)
USER=kubevirt-bot
EMAIL=kubevirtbot@redhat.com

generate_config(){
    local prow_config=${PROJECT_INFRA_ROOT}/github/ci/prow-deploy/kustom/base/configs/current/config/config.yaml
    local job_config=${PROJECT_INFRA_ROOT}/github/ci/prow-deploy/files/jobs
    local testgrid_dir=${TEST_INFRA_ROOT}/config/testgrids
    local testgrid_subdir=kubevirt

    /configurator \
        --prow-config "${prow_config}" \
        --prow-job-config "${job_config}" \
        --output-yaml \
        --yaml "${TESTGRID_GEN_CONFIG}" \
        --oneshot \
        --output "${testgrid_dir}/${testgrid_subdir}/gen-config.yaml"

    cp "${testgrid_dir}/${testgrid_subdir}/gen-config.yaml" "${TESTGRID_CONFIG}"

    git add --all
}

run_tests(){
    (
        ${PROJECT_INFRA_ROOT}/hack/create_bazel_cache_rcs.sh
        cd ${TEST_INFRA_ROOT}
        bazel test //config/tests/... //hack:verify-spelling
    )
}

ensure_git_config() {
    echo "Checking Git Config"
    git config user.name ${USER}
    git config user.email ${EMAIL}

    git config user.name &>/dev/null && git config user.email &>/dev/null && return 0
    echo "ERROR: git config user.name, user.email unset. No defaults provided" >&2
    return 1
}

upload_config(){
    gcloud auth activate-service-account --key-file=/etc/gcs/service-account.json
    gsutil cp ${TESTGRID_CONFIG} gs://kubevirt-prow/testgrid/config.yaml
}

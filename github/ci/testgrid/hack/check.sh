#!/bin/bash

set -euo pipefail

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_INFRA_ROOT=$(readlink -f --canonicalize ${BASEDIR}/../../../..)
if [ -d ${PROJECT_INFRA_ROOT}/../../kubernetes/test-infra ]; then
    TEST_INFRA_ROOT=$(readlink -f --canonicalize ${PROJECT_INFRA_ROOT}/../../kubernetes/test-infra)
fi

generate_config(){
    local prow_config=${PROJECT_INFRA_ROOT}/github/ci/prow-deploy/kustom/base/configs/current/config/config.yaml
    local job_config=${PROJECT_INFRA_ROOT}/github/ci/prow-deploy/files/jobs
    local testgrid_dir=${TEST_INFRA_ROOT}/config/testgrids
    local testgrid_subdir=kubevirt
    local testgrid_gen_config=$(readlink -f --canonicalize ${BASEDIR}/../gen-config.yaml)
    local testgrid_default=$(readlink -f --canonicalize ${BASEDIR}/../default.yaml)

    mkdir -p "${testgrid_dir}/${testgrid_subdir}"

    /var/run/ko/configurator \
        --prow-config "${prow_config}" \
        --prow-job-config "${job_config}" \
        --output-yaml \
        --yaml "${testgrid_gen_config}" \
        --default "${testgrid_default}" \
        --oneshot \
        --output "${testgrid_dir}/${testgrid_subdir}/gen-config.yaml"
}

run_tests(){
    (
        ${PROJECT_INFRA_ROOT}/hack/create_bazel_cache_rcs.sh
        cd ${TEST_INFRA_ROOT}
        bazel test --test_output=all --test_verbose_timeout_warnings $(bazel query //config/tests/testgrids/...)
        bazel test --test_output=all --test_verbose_timeout_warnings //hack:verify-spelling
    )
}

main(){
    generate_config "${@}"

    run_tests "${@}"
}

main "${@}"

#!/bin/bash

set -euo pipefail

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_INFRA_ROOT=$(readlink -f --canonicalize ${BASEDIR}/../../../..)
eval $(gimme ${GIMME_GO_VERSION})

if [ -d ${PROJECT_INFRA_ROOT}/../../kubernetes/test-infra ]; then
    TEST_INFRA_ROOT=$(readlink -f --canonicalize ${PROJECT_INFRA_ROOT}/../../kubernetes/test-infra)
fi

build_configurator() {
	cd ${TEST_INFRA_ROOT}/testgrid/cmd/configurator/ && go build
	mv ./configurator /configurator && cd ${PROJECT_INFRA_ROOT}
}

generate_config(){
    local prow_config=${PROJECT_INFRA_ROOT}/github/ci/prow-deploy/kustom/base/configs/current/config/config.yaml
    local job_config=${PROJECT_INFRA_ROOT}/github/ci/prow-deploy/files/jobs
    local testgrid_dir=${TEST_INFRA_ROOT}/config/testgrids
    local testgrid_subdir=kubevirt
    local testgrid_gen_config=$(readlink -f --canonicalize ${BASEDIR}/../gen-config.yaml)
    local testgrid_default=$(readlink -f --canonicalize ${BASEDIR}/../default.yaml)

    mkdir -p "${testgrid_dir}/${testgrid_subdir}"

    /configurator \
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
        cd ${TEST_INFRA_ROOT}/config/tests/testgrids/
	go test config_test.go
    )
}

main(){

    build_configurator

    generate_config "${@}"

    run_tests "${@}"
}

main "${@}"

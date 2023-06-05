#!/bin/bash

set -e

main(){
    current_dir=$(dirname "$0")
    project_infra_root=$(readlink -f "${current_dir}/../../../..")

    base_dir=${project_infra_root}/github/ci/prow-deploy

    cd ${base_dir}

    # preserve exit status for later to capture the artifacts in any case
    set +e
    molecule test
    retval=$?
    set -e

    tmp=$(mktemp -d)
    docker cp instance:$ARTIFACTS $tmp
    cp -ar $tmp/artifacts/* $ARTIFACTS

    if [ $retval -ne 0 ]; then
        exit $retval
    fi
}

main "${@}"

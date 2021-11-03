#!/bin/bash

set -e

main(){
    current_dir=$(dirname "$0")
    project_infra_root=$(readlink -f "${current_dir}/../../../..")

    base_dir=${project_infra_root}/github/ci/prow-deploy

    cd ${base_dir}

    molecule test
    tmp=$(mktemp -d)
    docker cp instance:$ARTIFACTS $tmp
    cp -ar $tmp/artifacts/* $ARTIFACTS
}

main "${@}"

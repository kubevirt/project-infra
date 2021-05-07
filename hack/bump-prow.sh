#!/bin/bash

set -euxo pipefail

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_INFRA_ROOT=$(readlink --canonicalize ${BASEDIR}/..)
PROJECT_INFRA_MANIFESTS_ROOT=${PROJECT_INFRA_ROOT}/github/ci/prow-deploy/kustom/base
TEST_INFRA_ROOT=$(readlink --canonicalize ${PROJECT_INFRA_ROOT}/../../kubernetes/test-infra)
TEST_INFRA_MANIFESTS_ROOT=${TEST_INFRA_ROOT}/config/prow/cluster

copy_files(){
    local target_files=$(yq r ${PROJECT_INFRA_MANIFESTS_ROOT}/kustomization.yaml resources | grep test_infra | sed 's/^- \(.*\)/\1/')

    for base_target_file in ${target_files}; do
        local target_file_name=$(basename ${base_target_file})
        local target_file=${PROJECT_INFRA_MANIFESTS_ROOT}/manifests/test_infra/current/${target_file_name}
        local source_file=${TEST_INFRA_MANIFESTS_ROOT}/$(basename $target_file)

        cp ${source_file} ${target_file}
    done
}

get_latest_prow_tag(){
    echo $(grep gcr.io/k8s-prow/ ${TEST_INFRA_MANIFESTS_ROOT}/prow_controller_manager_deployment.yaml | cut -d ':' -f 3)
}

bump_utility_images(){
    local latest_prow_tag=$(get_latest_prow_tag)

    if [ -z "${latest_prow_tag}" ]; then
        echo "Could not find latest prow tag"
        exit 1
    fi

    echo latest_prow_tag: $latest_prow_tag

    local utility_images=(clonerefs initupload entrypoint sidecar)

    for utility_image in ${utility_images[@]}; do
        sed -i "s!${utility_image}: \"gcr.io/k8s-prow/${utility_image}:.*\"!${utility_image}: \"gcr.io/k8s-prow/${utility_image}:${latest_prow_tag}\"!" ${PROJECT_INFRA_MANIFESTS_ROOT}/configs/current/config/config.yaml
    done
}

main(){
    copy_files

    bump_utility_images
}

main "${@}"

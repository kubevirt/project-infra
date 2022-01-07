#!/bin/bash

set -euxo pipefail

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_INFRA_ROOT=$(readlink --canonicalize ${BASEDIR}/..)
PROJECT_INFRA_MANIFESTS_ROOT=${PROJECT_INFRA_ROOT}/github/ci/prow-deploy/kustom/base
TEST_INFRA_ROOT=$(readlink --canonicalize ${PROJECT_INFRA_ROOT}/../../kubernetes/test-infra)
TEST_INFRA_MANIFESTS_ROOT=${TEST_INFRA_ROOT}/config/prow/cluster

copy_files(){
    curl -Lo ./yq https://github.com/mikefarah/yq/releases/download/3.4.1/yq_linux_amd64 && \
        chmod a+x ./yq && \
        mv ./yq /usr/local/bin

    local target_files=$(yq r ${PROJECT_INFRA_MANIFESTS_ROOT}/kustomization.yaml resources | grep test_infra | sed 's/^- \(.*\)/\1/')

    for base_target_file in ${target_files}; do
        local target_dir_name=$(dirname ${base_target_file})
        local target_file=${PROJECT_INFRA_MANIFESTS_ROOT}/${base_target_file}

        mkdir -p ${PROJECT_INFRA_MANIFESTS_ROOT}/${target_dir_name}

        local source_file=${TEST_INFRA_MANIFESTS_ROOT}/${base_target_file/manifests\/test_infra\/current\//}

        cp ${source_file} ${target_file}
    done
}

get_latest_prow_tag(){
    echo $(grep gcr.io/k8s-prow/ ${TEST_INFRA_MANIFESTS_ROOT}/prow_controller_manager_deployment.yaml | cut -d ':' -f 3)
}

bump_utility_images(){
    local latest_prow_tag=$1

    local utility_images=(clonerefs initupload entrypoint sidecar)

    for utility_image in ${utility_images[@]}; do
        sed -i "s!${utility_image}: \"gcr.io/k8s-prow/${utility_image}:.*\"!${utility_image}: \"gcr.io/k8s-prow/${utility_image}:${latest_prow_tag}\"!" ${PROJECT_INFRA_MANIFESTS_ROOT}/configs/current/config/config.yaml
    done
}

bump_exporter(){
    local latest_prow_tag=$1

    sed -i "s!image: gcr.io/k8s-prow/exporter:.*!image: gcr.io/k8s-prow/exporter:${latest_prow_tag}!" ${PROJECT_INFRA_ROOT}/github/ci/prow-deploy/kustom/overlays/ibmcloud-production/resources/prow-exporter-deployment.yaml
}

bump_base_manifests_local_images(){
    local latest_prow_tag=$1

    find ${PROJECT_INFRA_ROOT}/github/ci/prow-deploy/kustom/base/manifests/local -type f -name '*.yaml' | xargs sed -i "s!image: gcr.io/k8s-prow/\(.*\):.*!image: gcr.io/k8s-prow/\\1:${latest_prow_tag}!"
}

main(){
    copy_files

    local latest_prow_tag=$(get_latest_prow_tag)
    if [ -z "${latest_prow_tag}" ]; then
        echo "Could not find latest prow tag"
        exit 1
    fi

    echo latest_prow_tag: $latest_prow_tag

    bump_utility_images "${latest_prow_tag}"
    bump_exporter "${latest_prow_tag}"
    bump_base_manifests_local_images "${latest_prow_tag}"
}

main "${@}"

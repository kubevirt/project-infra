#!/bin/bash

# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright the KubeVirt Authors.

set -eu -o pipefail

base_dir=$(realpath "$(dirname "${BASH_SOURCE[0]}")/..")
project_infra_root=${base_dir%/*/*/*}

main(){
    source ${project_infra_root}/hack/manage-secrets.sh
    decrypt_secrets

    generate_consolidated_kubeconfig
    populate_secrets

    cleanup_secrets

    # run playbook
    cd "${base_dir}"
    export GIT_ASKPASS="${project_infra_root}"/hack/git-askpass.sh
    cat << EOF > inventory
[local]
localhost ansible_connection=local
EOF
    ANSIBLE_ROLES_PATH="${base_dir%/*}" ansible-playbook -i inventory prow-deploy.yaml
}

generate_consolidated_kubeconfig(){
    local prow_build_clusters kubeconfig_list kubeconfig_tmp

    prow_build_clusters=(
        kubevirt-prow-control-plane
        amd-workloads
        prow-arm64-workloads
        prow-hyperv-workloads
        prow-s390x-workloads
        prow-workloads
    )

    kubeconfig_list=$(
        printf "${secrets_repo_dir}/secrets/kubeconfigs/%s:" "${prow_build_clusters[@]}"
    )

    kubeconfig_tmp=$(mktemp)
    KUBECONFIG="${kubeconfig_list}" kubectl config view --flatten --raw > "${kubeconfig_tmp}"

    mkdir -p ${base_dir}/vars/${DEPLOY_ENVIRONMENT}

    yq '
      .kubeconfig |= load_str("'"${kubeconfig_tmp}"'")
    ' "${secrets_repo_dir}"/main.yml > "${base_dir}/vars/${DEPLOY_ENVIRONMENT}"/secrets.yml

    rm -f "${kubeconfig_tmp}"
}

populate_secrets(){
    local secrets_dir="${base_dir}/kustom/overlays/${DEPLOY_ENVIRONMENT}/secrets"

    mkdir -p "${secrets_dir}"/github

    install -Dm 400 "${secrets_repo_dir}"/secrets/prow/github/app-id "${secrets_dir}"/github/
    install -Dm 400 "${secrets_repo_dir}"/secrets/prow/github/app-secret "${secrets_dir}"/github/
    install -Dm 400 "${secrets_repo_dir}"/secrets/prow/github/bot-token "${secrets_dir}"/github/
}

main "${@}"

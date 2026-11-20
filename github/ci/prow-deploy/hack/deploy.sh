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

    local secrets_dir="${base_dir}/kustom/overlays/${DEPLOY_ENVIRONMENT}/secrets"
    mkdir -p "${secrets_dir}"
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
    local prow_build_clusters kubeconfig_list

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

    KUBECONFIG="${kubeconfig_list}" kubectl config view --flatten --raw > "${secrets_dir}/kubeconfig"

}

populate_secrets(){
    install -Dm 400 "${secrets_repo_dir}"/tokens/product-keys/windows/sysprep-001 "${secrets_dir}"/win-sysprep-001

    install -Dm 400 "${secrets_repo_dir}"/tokens/fossa/kubevirt/push "${secrets_dir}"/kubevirtci-fossa-token

    install -Dm 400 "${secrets_repo_dir}"/tokens/coveralls/kubevirt/repo "${secrets_dir}"/kubevirtci-coveralls-token
    install -Dm 400 "${secrets_repo_dir}"/tokens/coveralls/containerized-data-importer/repo "${secrets_dir}"/containerized-data-importer-coveralls-token

    mkdir -p "${secrets_dir}"/github

    install -Dm 400 "${secrets_repo_dir}"/secrets/prow/cookie "${secrets_dir}"/
    install -Dm 400 "${secrets_repo_dir}"/secrets/prow/github/app-id "${secrets_dir}"/github/
    install -Dm 400 "${secrets_repo_dir}"/secrets/prow/github/app-secret "${secrets_dir}"/github/
    install -Dm 400 "${secrets_repo_dir}"/secrets/prow/github/bot-token "${secrets_dir}"/github/
    install -Dm 400 "${secrets_repo_dir}"/secrets/prow/github/client-id "${secrets_dir}"/github/
    install -Dm 400 "${secrets_repo_dir}"/secrets/prow/github/client-secret "${secrets_dir}"/github/
    install -Dm 400 "${secrets_repo_dir}"/secrets/prow/github/hmac-token "${secrets_dir}"/github/
    install -Dm 400 "${secrets_repo_dir}"/secrets/slack/kubevirtci-token "${secrets_dir}"/slack/token

    mkdir -p "${secrets_dir}"/kubevirtci-quay-credential
    install -Dm 400 "${secrets_repo_dir}"/creds/quay.io/kubevirtcibot/username "${secrets_dir}"/kubevirtci-quay-credential/
    install -Dm 400 "${secrets_repo_dir}"/creds/quay.io/kubevirtcibot/password "${secrets_dir}"/kubevirtci-quay-credential/
    install -Dm 400 "${secrets_repo_dir}"/creds/quay.io/kubevirtcibot/token "${secrets_dir}"/kubevirtci-quay-credential/

    mkdir -p "${secrets_dir}"/quay-kubevirt-push
    install -Dm 400 "${secrets_repo_dir}"/creds/quay.io/kubevirt+prow-push/username "${secrets_dir}"/quay-kubevirt-push/
    install -Dm 400 "${secrets_repo_dir}"/creds/quay.io/kubevirt+prow-push/password "${secrets_dir}"/quay-kubevirt-push/

    mkdir -p "${secrets_dir}"/quay-kubevirtci-push
    install -Dm 400 "${secrets_repo_dir}"/creds/quay.io/kubevirtci+prow-push/username "${secrets_dir}"/quay-kubevirtci-push/
    install -Dm 400 "${secrets_repo_dir}"/creds/quay.io/kubevirtci+prow-push/password "${secrets_dir}"/quay-kubevirtci-push/

    mkdir -p "${secrets_dir}"/quay-containerdisks-push
    install -Dm 400 "${secrets_repo_dir}"/creds/quay.io/containerdisks+prow-push/username "${secrets_dir}"/quay-containerdisks-push/
    install -Dm 400 "${secrets_repo_dir}"/creds/quay.io/containerdisks+prow-push/password "${secrets_dir}"/quay-containerdisks-push/

    install -Dm 400 "${secrets_repo_dir}"/tokens/github/kubevirt-commenter-bot/prow-2026-11 "${secrets_dir}"/commenter-oauth-token
    install -Dm 400 "${secrets_repo_dir}"/tokens/google/service-accounts/bazel-cache/bazel-cache-sa.json "${secrets_dir}"/prow/google/service-accounts/bazel-cache/bazel-cache-sa.json
    install -Dm 400 "${secrets_repo_dir}"/tokens/google/service-accounts/prow/prow-sa.json "${secrets_dir}"/service-account.json
    install -Dm 400 "${secrets_repo_dir}"/tokens/google/service-accounts/virtualization-ci/virtualization-ci-sa.json "${secrets_dir}"/virtci-service-account.json

    local components_dir="${base_dir}/kustom/components"
    install -Dm 400 "${secrets_repo_dir}"/secrets/docker-mirror-proxy/ca.crt "${components_dir}"/docker-mirror-proxy/base/configs/
    install -Dm 400 "${secrets_repo_dir}"/secrets/docker-mirror-proxy/ca.key "${components_dir}"/docker-mirror-proxy/base/secrets/
}

main "${@}"

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

unset "${!KUBERNETES@}"

base_dir=$(realpath "$(dirname "${BASH_SOURCE[0]}")/..")
project_infra_root=${base_dir%/*/*/*}

DEPLOY_ENVIRONMENT=${DEPLOY_ENVIRONMENT:-kubevirtci-testing}

main(){
    source ${project_infra_root}/hack/manage-secrets.sh
    decrypt_secrets

    populate_secrets

    cleanup_secrets

    KUBEVIRT_DIR=${KUBEVIRT_DIR:-/home/prow/go/src/github.com/kubevirt/kubevirt}
    export KUBEVIRT_MEMORY_SIZE=16384M

    cd $KUBEVIRT_DIR && export KUBEVIRT_PROVIDER=$(find kubevirtci/cluster-up/cluster/k8s-1* -maxdepth 0 -type d -printf '%f\n' | sort -r |  head -1)

    make cluster-up

    export KUBECONFIG=$(./kubevirtci/cluster-up/kubeconfig.sh)

    kubectl create ns kubevirt-prow && kubectl create ns kubevirt-prow-jobs

    kubectl label node node01 ci.kubevirt.io/cachenode=true ingress-ready=true

    POD_NAME=$KUBEVIRT_PROVIDER-node01

    NODE_POD_IP=$(podman inspect $POD_NAME -f '{{ .NetworkSettings.IPAddress }}')

    echo "$NODE_POD_IP gcsweb.prowdeploy.ci deck.prowdeploy.ci" >> /etc/hosts

    cd "${base_dir}"

    cat << EOF > inventory
[local]
localhost ansible_connection=local
EOF
    ANSIBLE_ROLES_PATH="${base_dir%/*}" ansible-playbook -i inventory --extra-vars project_infra_root="${project_infra_root}"  --extra-vars kubeconfig_path="${KUBECONFIG}" prow-deploy.yaml
}

populate_secrets(){
    local secrets_dir="${base_dir}/kustom/overlays/${DEPLOY_ENVIRONMENT}/secrets"

    mkdir -p "${secrets_dir}"/github

    install -Dm 400 "${secrets_repo_dir}"/secrets/prow-staging/github/app-id "${secrets_dir}"/github/
    install -Dm 400 "${secrets_repo_dir}"/secrets/prow-staging/github/app-secret "${secrets_dir}"/github/
    install -Dm 400 "${secrets_repo_dir}"/secrets/prow-staging/github/bot-token "${secrets_dir}"/github/
}

main "$@"

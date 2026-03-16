#!/bin/bash

set -eu -o pipefail

main(){
    current_dir=$(dirname "$0")
    project_infra_root=$(readlink -f "${current_dir}/../../../..")

    base_dir=${project_infra_root}/github/ci/prow-deploy

    # TODO: remove yq installation after prow-deploy image tag bump
    curl -fsSLo ./yq https://github.com/mikefarah/yq/releases/download/v4.47.1/yq_linux_amd64
    chmod +x ./yq && mv ./yq /usr/local/bin/yq

    source ${project_infra_root}/hack/manage-secrets.sh
    decrypt_secrets

    mkdir -p ${base_dir}/vars/${DEPLOY_ENVIRONMENT}
    mv "${secrets_repo_dir}"/main.yml ${base_dir}/vars/${DEPLOY_ENVIRONMENT}/secrets.yml

    cleanup_secrets

    # run playbook
    cd ${base_dir}
    export GIT_ASKPASS=${project_infra_root}/hack/git-askpass.sh
    cat << EOF > inventory
[local]
localhost ansible_connection=local
EOF
    ANSIBLE_ROLES_PATH=$(pwd)/.. ansible-playbook -i inventory prow-deploy.yaml
}

main "${@}"

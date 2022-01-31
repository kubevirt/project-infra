#!/bin/bash

set -e

main(){
    current_dir=$(dirname "$0")
    project_infra_root=$(readlink -f "${current_dir}/../../../..")

    base_dir=${project_infra_root}/github/ci/prow-deploy

    source ${project_infra_root}/hack/manage-secrets.sh
    decrypt_secrets

    mkdir -p ${base_dir}/vars/${DEPLOY_ENVIRONMENT}
    mv main.yml ${base_dir}/vars/${DEPLOY_ENVIRONMENT}/secrets.yml

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

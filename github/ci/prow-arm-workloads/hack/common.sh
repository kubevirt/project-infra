#!/bin/bash

PROJECT_PATH=$(realpath $(dirname "$0")/../../../..)
BASE_DIR=${PROJECT_PATH}/github/ci/prow-arm-workloads

setup(){
    cd ${PROJECT_PATH}

    source ./hack/manage-secrets.sh
    decrypt_secrets
    # ssh key
    mkdir -p ~/.ssh && chmod 0700 ~/.ssh
    extract_secret 'prowARM.sshKey' ~/.ssh/id_rsa
    chmod 0600 ~/.ssh/id_rsa
    printf "\n" >> ~/.ssh/id_rsa
    # ansible config files
    extract_secret 'prowARM.hostsYml' ${BASE_DIR}/inventory/prow-arm-workloads/hosts.yml

    mkdir -p /etc/ansible
    cat <<EOF > /etc/ansible/ansible.cfg
[defaults]
host_key_checking = False
EOF
}

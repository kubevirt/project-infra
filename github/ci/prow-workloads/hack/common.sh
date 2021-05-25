#!/bin/bash

PROJECT_PATH=$(realpath $(dirname "$0")/../../../..)
BASE_DIR=${PROJECT_PATH}/github/ci/prow-workloads

setup(){
    cd ${PROJECT_PATH}

    source ./hack/manage-secrets.sh
    decrypt_secrets
    # ssh key
    mkdir -p ~/.ssh && chmod 0700 ~/.ssh
    extract_secret 'prowWorkloads.sshKey' ~/.ssh/id_ed25519
    chmod 0600 ~/.ssh/id_ed25519
    # ansible config files
    extract_secret 'prowWorkloads.hostsYml' ${BASE_DIR}/inventory/prow-workloads/hosts.yml
    extract_secret 'prowWorkloads.groupVarsAllYml' ${BASE_DIR}/inventory/prow-workloads/group_vars/all/all.yml
}

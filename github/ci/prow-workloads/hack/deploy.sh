#!/bin/bash

main(){
    (
        PROJECT_PATH=$(realpath $(dirname "$0")/../../../..)

        cd ${PROJECT_PATH}

        base_dir=github/ci/prow-workloads

        source ./hack/manage-secrets.sh
        decrypt_secrets
        # ssh key
        mkdir -p ~/.ssh && chmod 0700 ~/.ssh
        extract_secret 'prowWorkloads.ssKey' ~/.ssh/id_ed25519
        chmod 0600 ~/.ssh/id_ed25519
        # ansible config files
        extract_secret 'prowWorkloads.hostsYml' ${base_dir}/inventory/prow-workloads/hosts.yml
        extract_secret 'prowWorkloads.groupVarsAllYml' ${base_dir}/inventory/prow-workloads/group_vars/all/all.yml

        cd ${base_dir}

        ansible-playbook -i inventory/prow-workloads/hosts.yaml  --become --become-user=root --private-key ~/.ssh/id_ed25519 cluster.yml
    )
}

main "$@"

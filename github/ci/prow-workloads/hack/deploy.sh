#!/bin/bash

source $(dirname "$0")/common.sh

main(){
    setup

    cd ${BASE_DIR}

    ansible-playbook -i inventory/prow-workloads/hosts.yaml  --become --become-user=root --private-key ~/.ssh/id_ed25519 /kubespray/cluster.yml
}

main "$@"

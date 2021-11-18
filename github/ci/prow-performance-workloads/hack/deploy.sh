#!/bin/bash

source $(dirname "$0")/common.sh

main(){
    setup

    cd ${BASE_DIR}

    ansible-playbook -i inventory/prow-performance-workloads/hosts.yml  --become --become-user=root --private-key ~/.ssh/id_rsa /kubespray/cluster.yml
}

main "$@"

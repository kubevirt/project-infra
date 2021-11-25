#!/bin/bash

source $(dirname "$0")/common.sh

main(){
    setup

    cd /kubespray
    
    cp -r ${BASE_DIR}/inventory/prow-performance-workloads inventory/

    ansible-playbook -i inventory/prow-performance-workloads/hosts.yml  --become --become-user=root --private-key ~/.ssh/id_rsa cluster.yml
}

main "$@"

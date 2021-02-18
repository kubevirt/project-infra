#!/bin/bash

set -euo pipefail

decrypt_secrets(){
    target_dir=$(mktemp -d)
    git clone https://kubevirt-bot@github.com/kubevirt/secrets ${target_dir}
    gpg --allow-secret-key-import --import /etc/pgp/token
    gpg --decrypt ${target_dir}/secrets.tar.asc > secrets.tar
    tar -xvf secrets.tar
    rm secrets.tar
    if [ ! -f $(pwd)/main.yml ]; then
        echo "Secrets file not present after unencrypting and unpacking"
        exit 1
    fi
}

extract_secret(){
    local key="${1}"
    local path="${2}"

    mkdir -p $(dirname "${path}")
    # only remove new line at the end
    yq r main.yml "${key}" | awk 'NR>1{print PREV} {PREV=$0} END{printf("%s",$0)}' > "${path}"
}

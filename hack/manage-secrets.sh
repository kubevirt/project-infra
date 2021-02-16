#!/bin/bash

set -euo pipefail

decrypt_secrets(){
    local github_token_path="${1}"

    target_dir=$(mktemp -d)
    git clone https://kubevirt-bot:$(cat ${github_token_path})@github.com/kubevirt/secrets ${target_dir}
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

    yq r main.yml "${key}" | tr -d "\n" > "${path}"
}

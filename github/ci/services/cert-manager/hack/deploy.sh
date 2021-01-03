#!/bin/bash

set -eo pipefail

install_bazelisk(){
    curl --fail -L https://github.com/bazelbuild/bazelisk/releases/download/v1.7.4/bazelisk-linux-amd64 --output ./bazelisk
    chmod a+x ./bazelisk && mv ./bazelisk /usr/local/bin/bazelisk
}

main(){
    local environment=${1}

    # deploy base resources
    bazelisk run //github/ci/services/cert-manager:${environment}-base.apply

    # wait for rollouts done
    bazelisk run //github/ci/services/common/k8s/cmd/wait -- -namespace cert-manager -selector cert-manager-cainjector -kind deployment
    bazelisk run //github/ci/services/common/k8s/cmd/wait -- -namespace cert-manager -selector cert-manager-webhook -kind deployment


    # deploy issuers
    bazelisk run //github/ci/services/cert-manager:${environment}-issuers.apply
}

main "${@}"

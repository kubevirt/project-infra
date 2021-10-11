#!/bin/bash

set -eo pipefail

main(){
    local environment=${1}

    # deploy rbac
    bazelisk run //github/ci/services/cert-manager:${environment}-rbac.apply

    # deploy base resources
    bazelisk run //github/ci/services/cert-manager:${environment}-base.apply

    # wait for rollouts done
    bazelisk run //github/ci/services/common/k8s/cmd/wait -- -namespace cert-manager -selector cert-manager-cainjector -kind deployment
    bazelisk run //github/ci/services/common/k8s/cmd/wait -- -namespace cert-manager -selector cert-manager-webhook -kind deployment

    # deploy issuers
    bazelisk run //github/ci/services/cert-manager:${environment}-issuers.apply
}

main "${@}"

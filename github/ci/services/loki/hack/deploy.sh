#!/bin/bash

set -eo pipefail

main(){
    local environment=${1}

    bazelisk run //github/ci/services/loki:${environment}.apply

    bazelisk run //github/ci/services/common/k8s/cmd/wait -- -namespace monitoring -selector loki -kind statefulset
}

main "${@}"

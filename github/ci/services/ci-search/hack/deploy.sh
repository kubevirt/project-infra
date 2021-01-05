#!/bin/bash

set -eo pipefail

main(){
    local environment=${1}

    bazelisk run //github/ci/services/ci-search:${environment}.apply

    bazelisk run //github/ci/services/common/k8s/cmd/wait -- -namespace ci-search -selector search -kind statefulset
}

main "${@}"

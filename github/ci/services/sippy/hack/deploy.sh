#!/bin/bash

set -eo pipefail

main(){
    local environment=${1}

    bazelisk run //github/ci/services/sippy:${environment}.apply

    bazelisk run //github/ci/services/common/k8s/cmd/wait -- -namespace sippy -selector sippy -kind statefulset
}

main "${@}"

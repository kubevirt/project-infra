#!/bin/bash

set -eo pipefail

main(){
    local environment=${1}

    bazelisk run //github/ci/services/kubot:${environment}.apply

    bazelisk run //github/ci/services/common/k8s/cmd/wait -- -namespace kubot -selector kubot -kind deployment
}

main "${@}"

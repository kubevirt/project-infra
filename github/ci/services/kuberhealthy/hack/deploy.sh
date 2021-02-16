#!/bin/bash

set -eo pipefail

main(){
    local environment=${1}

    bazelisk run //github/ci/services/kuberhealthy:${environment}-crds.apply

    bazelisk run //github/ci/services/common/k8s/cmd/wait -- -selector khchecks.comcast.github.io -kind crd

    bazelisk run //github/ci/services/kuberhealthy:${environment}.apply

    bazelisk run //github/ci/services/common/k8s/cmd/wait -- -namespace kuberhealthy -selector kuberhealthy -kind deployment
}

main "${@}"

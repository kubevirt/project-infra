#!/bin/bash

set -euxo pipefail

GITHUB_TOKEN_PATH="${1:-/etc/github}"
if [ ! -d "$GITHUB_TOKEN_PATH" ]; then
    echo "$GITHUB_TOKEN_PATH is not a directory, should contain the oauth for github bot account"
    exit 1
fi

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_INFRA_ROOT=$(readlink --canonicalize ${BASEDIR}/..)

autobump() {
    relative_config_path="$1"
    # the below is necessary since running the autobumper inside a pod fails because of a failing git command
    (
        podman run -v ${PROJECT_INFRA_ROOT}/:/config:z -v ${GITHUB_TOKEN_PATH}:/etc/github -it gcr.io/k8s-prow/generic-autobumper:v20240611-924d31f0c --config /config/${relative_config_path} --skip-pullrequest
    ) || true
}

main(){
    autobump github/ci/prow-deploy/prow-autobump-config.yaml
    autobump github/ci/prow-deploy/prow-job-autobump-config.yaml
}

main "${@}"

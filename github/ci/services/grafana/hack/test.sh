#!/bin/bash

set -ex

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_INFRA_ROOT=$(readlink --canonicalize ${BASEDIR}/../../../../../)

cd "$PROJECT_INFRA_ROOT"

export KUBECONFIG="${KUBECONFIG}"

./github/ci/services/grafana/hack/install.sh testing
./github/ci/services/grafana/hack/deploy.sh testing

kubectl label -n monitoring svc/grafana-service app=grafana

# sometimes the service is not yet ready therefore the port-forwarding will fail on the
# initial try
retries=3
while ! env KUBECONFIG=$KUBECONFIG go test -v ./github/ci/services/grafana/e2e/e2e_test.go ; do
    sleep 5
    retries=$((retries-1))
    if [ $retries -le 0 ]; then
        echo "test failed"
        exit 1
    fi
done

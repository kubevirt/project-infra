#!/bin/bash

set -ex

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_INFRA_ROOT=$(readlink --canonicalize ${BASEDIR}/../../../../../)

cd $PROJECT_INFRA_ROOT

KUBEVIRT_DIR=${KUBEVIRT_DIR:-/home/prow/go/src/github.com/kubevirt/kubevirt}

cd $KUBEVIRT_DIR && make cluster-up

export KUBECONFIG=$(./kubevirtci/cluster-up/kubeconfig.sh)

cd "$PROJECT_INFRA_ROOT"

kubectl create ns monitoring

helm upgrade --namespace monitoring -i grafana-operator oci://ghcr.io/grafana-operator/helm-charts/grafana-operator --version v5.4.1

./github/ci/services/grafana/hack/deploy.sh testing

kubectl label -n monitoring svc/grafana-service app=grafana

bazelisk test //github/ci/services/grafana/e2e:go_default_test --test_env=KUBECONFIG=$KUBECONFIG --test_output=all --test_arg=-test.v


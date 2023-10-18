#!/bin/bash

set -ex

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_INFRA_ROOT=$(readlink --canonicalize ${BASEDIR}/../../../../../)

cd $PROJECT_INFRA_ROOT

cat <<EOF | kind create cluster --image quay.io/kubevirtci/kindest-node:v1.27.1 --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
EOF

kubectl create ns monitoring

# Install ingress
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

# Allow time for ingress controller to start
sleep 20

helm upgrade --namespace monitoring -i grafana-operator oci://ghcr.io/grafana-operator/helm-charts/grafana-operator --version v5.4.1

./github/ci/services/grafana/hack/deploy.sh testing

kubectl label -n monitoring svc/grafana-service app=grafana

bazelisk test //github/ci/services/grafana/e2e:go_default_test --test_output=all --test_arg=-test.v


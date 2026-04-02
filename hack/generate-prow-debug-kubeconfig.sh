#!/usr/bin/env bash
# based on
# https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/#to-create-additional-api-tokens
# we create a new token for the service account from which we generate a new KUBECONFIG
# we use cluster and context definitions from the currently used KUBECONFIG

set -x
set -euo pipefail

function usage {
    cat <<EOF
usage: $0 [--help] [--include-kubevirt-prow] <token-name>

    Generate a readonly KUBECONFIG for debugging Prow clusters.

    Creates a service account token for the prow-debug serviceaccount on each
    cluster and writes a standalone kubeconfig file to /tmp. The generated
    kubeconfig provides readonly access to prowjobs, pods, and pod logs in the
    kubevirt-prow-jobs namespace on both the kubevirt-prow-control-plane and
    prow-workloads clusters.

    Requires the prow-debug RBAC to be deployed first. If it is missing, run:
      hack/apply-prow-debug-rbac.sh

    Options:
      --help                    Show this help message and exit.
      --include-kubevirt-prow   Also create readonly access for the kubevirt-prow
                                namespace on the kubevirt-prow-control-plane cluster.

EOF
}

include_kubevirt_prow=false
if [ "${1:-}" == "--help" ]; then
    usage
    exit 0
fi
if [ "${1:-}" == "--include-kubevirt-prow" ]; then
    include_kubevirt_prow=true
    shift
fi

current_context=$(kubectl config current-context)

if [ ! "$#" -eq 1 ]; then
    usage
    exit 1
fi

token_name="prow-debug-$1"
token_namespace="kubevirt-prow-jobs"

clusters=( kubevirt-prow-control-plane prow-workloads )

for cluster in "${clusters[@]}"; do
    if ! kubectl --context "$cluster" -n "$token_namespace" get serviceaccount prow-debug &>/dev/null; then
        echo "ERROR: serviceaccount prow-debug not found in namespace $token_namespace on cluster $cluster"
        echo "Apply the RBAC manifest first by running: hack/apply-prow-debug-rbac.sh"
        exit 1
    fi
done

for cluster in "${clusters[@]}"; do
    kubectl config use-context "$cluster"

    kubectl delete --ignore-not-found=true -n "$token_namespace" secret "$token_name"

    kubectl create -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: $token_name
  namespace: $token_namespace
  annotations:
    kubernetes.io/service-account.name: prow-debug
type: kubernetes.io/service-account-token
EOF
done

if [ "$include_kubevirt_prow" == "true" ]; then
    if ! kubectl --context kubevirt-prow-control-plane -n kubevirt-prow get serviceaccount prow-debug &>/dev/null; then
        echo "ERROR: serviceaccount prow-debug not found in namespace kubevirt-prow on cluster kubevirt-prow-control-plane"
        echo "Apply the RBAC manifest first by running: hack/apply-prow-debug-rbac.sh"
        exit 1
    fi

    kubectl config use-context kubevirt-prow-control-plane

    kubectl delete --ignore-not-found=true -n kubevirt-prow secret "$token_name"

    kubectl create -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: $token_name
  namespace: kubevirt-prow
  annotations:
    kubernetes.io/service-account.name: prow-debug
type: kubernetes.io/service-account-token
EOF
fi

token_kubevirt_prow_control_plane=$(kubectl config use-context kubevirt-prow-control-plane 2>&1 > /dev/null && kubectl get secret "$token_name" -n "$token_namespace" -o yaml | yq -r '.data.token' | base64 -d)
token_prow_workloads=$(kubectl config use-context prow-workloads 2>&1 > /dev/null && kubectl get secret "$token_name" -n "$token_namespace" -o yaml | yq -r '.data.token' | base64 -d)

kubeconfig_clusters=$(yq '.clusters' "$KUBECONFIG")

kubevirt_prow_context=""
if [ "$include_kubevirt_prow" == "true" ]; then
    kubevirt_prow_context="- context:
    cluster: kubevirt-prow-control-plane
    namespace: kubevirt-prow
    user: prow-debug-kubevirt-prow-control-plane
  name: kubevirt-prow-control-plane-kubevirt-prow"
fi

cat <<EOF > "/tmp/kubeconfig_$token_name.yaml"
apiVersion: v1
clusters:
$kubeconfig_clusters
contexts:
- context:
    cluster: kubevirt-prow-control-plane
    namespace: $token_namespace
    user: prow-debug-kubevirt-prow-control-plane
  name: kubevirt-prow-control-plane
- context:
    cluster: prow-workloads
    namespace: $token_namespace
    user: prow-debug-prow-workloads
  name: prow-workloads
$kubevirt_prow_context
current-context: kubevirt-prow-control-plane
kind: Config
preferences: {}
users:
- name: prow-debug-kubevirt-prow-control-plane
  user:
    token: $token_kubevirt_prow_control_plane
- name: prow-debug-prow-workloads
  user:
    token: $token_prow_workloads
EOF

kubectl config use-context "$current_context"

echo "New config is ready, to use it, execute"
echo "export KUBECONFIG=/tmp/kubeconfig_$token_name.yaml"

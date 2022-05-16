#!/usr/bin/env bash
# based on
# https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/#to-create-additional-api-tokens
# we create a new token for the service account from which we generate a new KUBECONFIG
# we use cluster and context definitions from the currently used KUBECONFIG

set -euo pipefail

function usage {
    cat <<EOF
usage: $0 <token-name>

    Create a KUBECONFIG based on a newly created token for the prow-debug serviceaccount.

EOF
}

current_context=$(kubectl config current-context)

if [ ! "$#" -eq 1 ]; then
    usage
    exit 1
fi

token_name="prow-debug-$1"

clusters=( ibm-prow-jobs prow-workloads )

for cluster in "${clusters[@]}"; do
    kubectl config use-context "$cluster"

    kubectl delete --ignore-not-found=true secret "$token_name"

    kubectl create -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: $token_name
  annotations:
    kubernetes.io/service-account.name: prow-debug
type: kubernetes.io/service-account-token
EOF
done

token_ibm_prow_jobs=$(kubectl config use-context ibm-prow-jobs 2>&1 > /dev/null && kubectl get secret "$token_name" -o yaml | yq -r '.data.token' | base64 -d)
token_prow_workloads=$(kubectl config use-context prow-workloads 2>&1 > /dev/null && kubectl get secret "$token_name" -o yaml | yq -r '.data.token' | base64 -d)

kubeconfig_clusters=$(yq -y '.clusters' "$KUBECONFIG")

cat <<EOF > "/tmp/kubeconfig_$token_name.yaml"
apiVersion: v1
clusters:
$kubeconfig_clusters
contexts:
- context:
    cluster: ibm-cluster
    namespace: kubevirt-prow-jobs
    user: prow-debug-ibm-cluster
  name: ibm-prow-jobs
- context:
    cluster: prow-workloads-cluster
    namespace: kubevirt-prow-jobs
    user: prow-debug-prow-workloads-cluster
  name: prow-workloads
current-context: ibm-prow-jobs
kind: Config
preferences: {}
users:
- name: prow-debug-ibm-cluster
  user:
    token: $token_ibm_prow_jobs
- name: prow-debug-prow-workloads-cluster
  user:
    token: $token_prow_workloads
EOF

kubectl config use-context "$current_context"

echo "New config is ready, to use it, execute"
echo "export KUBECONFIG=/tmp/kubeconfig_$token_name.yaml"

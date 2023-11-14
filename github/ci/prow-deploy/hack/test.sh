#!/bin/bash

set -e

unset "${!KUBERNETES@}"

DEPLOY_ENVIRONMENT=${DEPLOY_ENVIRONMENT:-kubevirtci-testing}
CURRENT_DIR=$(dirname "$0")
PROJECT_INFRA_ROOT=$(readlink -f "${CURRENT_DIR}/../../../..")

BASE_DIR=${PROJECT_INFRA_ROOT}/github/ci/prow-deploy

export GIT_ASKPASS="${PROJECT_INFRA_ROOT}/hack/git-askpass.sh"

KUBEVIRT_DIR=${KUBEVIRT_DIR:-/home/prow/go/src/github.com/kubevirt/kubevirt}
export KUBEVIRT_MEMORY_SIZE=16384M
export KUBEVIRT_PROVIDER=k8s-1.27

cd $KUBEVIRT_DIR && make cluster-up

export KUBECONFIG=$(./cluster-up/kubeconfig.sh)

kubectl create ns kubevirt-prow && kubectl create ns kubevirt-prow-jobs

kubectl label node node01 ci.kubevirt.io/cachenode=true ingress-ready=true

POD_NAME=$KUBEVIRT_PROVIDER-node01
if [ "${CI}" == "true" ]; then
    POD_NAME=$JOB_NAME-node01
fi

NODE_POD_IP=$(podman inspect $POD_NAME -f '{{ .NetworkSettings.IPAddress }}')

echo "$NODE_POD_IP gcsweb.prowdeploy.ci deck.prowdeploy.ci" >> /etc/hosts

cd $BASE_DIR

cat << EOF > inventory
[local]
localhost ansible_connection=local
EOF

ANSIBLE_ROLES_PATH=$(pwd)/.. ansible-playbook -i inventory --extra-vars project_infra_root=$PROJECT_INFRA_ROOT --extra-vars kubeconfig_path=$KUBECONFIG prow-deploy.yaml

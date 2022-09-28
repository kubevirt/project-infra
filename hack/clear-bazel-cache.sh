#!/bin/bash

#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2022 Red Hat, Inc.
#
#

set -euxo pipefail

function usage() {
    cat <<EOF
usage: $0

    Clears the bazel cache by rescaling the service and deleting the data

    Needs to recreate the greenhouse service, thus it is required to be run from project-infra repo root directory.
EOF
}

function rescale_greenhouse_deployment() {
    kubectl scale deployment -n kubevirt-prow greenhouse --replicas=0
    kubectl rollout status -w deployment -n kubevirt-prow greenhouse
    wait_for_running_pods_number 0

    kubectl scale deployment -n kubevirt-prow greenhouse --replicas=1
    kubectl rollout status -w deployment -n kubevirt-prow greenhouse
    wait_for_running_pods_number 1
}

function remove_greenhouse_data() {
    greenhouse_pod_name=$(kubectl get pods --no-headers -n kubevirt-prow -l app=greenhouse --field-selector=status.phase=Running | head -1 | awk '{print $1}')
    kubectl exec "$greenhouse_pod_name" -n kubevirt-prow -- rm -rf /data/*
}

function wait_for_running_pods_number() {
    local running_pods_number="$1"
    while [[ $(kubectl get pods --no-headers -n kubevirt-prow -l app=greenhouse --field-selector=status.phase=Running | wc -l) -ne "$running_pods_number" ]]; do
        echo "number of running pods is $(kubectl get pods --no-headers -n kubevirt-prow -l app=greenhouse --field-selector=status.phase=Running | wc -l), desired $running_pods_number"
        sleep 3
    done
}

function main() {
    if [[ ! -f "$(pwd)/github/ci/prow-deploy/kustom/components/greenhouse/base/resources/service.yaml" ]]; then
        echo "service file for greenhouse not found in $(pwd)!"
        exit 1
    fi

    kubectl delete svc -n kubevirt-prow bazel-cache
    rescale_greenhouse_deployment
    remove_greenhouse_data
    rescale_greenhouse_deployment
    kubectl apply -f github/ci/prow-deploy/kustom/components/greenhouse/base/resources/service.yaml
}

main "$@"

#!/usr/bin/env bash
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
# Copyright the KubeVirt Authors.
#
#

set -e
set -u
set -o pipefail

set -x


WORK_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(cd "${WORK_DIR}/.." && pwd)"

function usage {
    cat <<EOF
usage: $0 [-d] <environment>
EOF
}

dry_run=false
while getopts "d" opt; do
    case "${opt}" in
        d )
            dry_run="true"
            shift
            ;;
        \? )
            usage
            exit 1
            ;;
    esac
done

environment="${1}"
[ -d "${BASE_DIR}/overlays/${environment}" ] || exit 1

cd "${BASE_DIR}"

if [ "$dry_run" = "true" ]; then
    kubectl kustomize "./overlays/${environment}"
    exit 0
fi

kubectl apply -k "./overlays/${environment}"

go run ../common/k8s/cmd/wait -namespace monitoring -selector grafana-deployment -kind deployment

kubectl wait --timeout 60s --for=jsonpath='{.status.stage}'=complete -n monitoring grafana grafana

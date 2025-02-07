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

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_INFRA_ROOT=$(readlink --canonicalize ${BASEDIR}/..)

function main() {
    if [ "$#" -eq 0 ]; then
        usage
        exit 1
    fi

    if [[ "$1" =~ -h ]]; then
        usage
        exit 0
    fi

    echo "$(extract_env_vars_to_export "$1" "$2")"
}

function extract_env_vars_to_export() {
    if [ -z "$1" ]; then
        usage
        exit 1
    fi

    job_type="${2:-periodics}"

    rg -l "$1" "${PROJECT_INFRA_ROOT}/github/ci/prow-deploy/files/jobs" | \
        xargs yq -r '.'"${job_type}"'[] | select(.name=="'"$1"'") | .spec.containers[0].env[] | "export "+.name+"="+.value+ " &&"' | \
        tr '\n' ' ' | \
        sed 's/\s&&\s*$//'
}

function usage() {
    cat << EOF
usage:
    $0 -h|--help
    $0 <job_name> [job-type(default: periodics)]

    Transforms the env var section of a prow job into a set of export commands and prints them out.

    Can then be used as input for eval like this:

    eval "\$(./hack/export-job-env-vars.sh periodic-kubevirt-e2e-k8s-1.31-sig-storage-root)"

    for presubmits

    ./hack/export-job-env-vars.sh pull-kubevirt-e2e-k8s-1.31-sig-compute 'presubmits."kubevirt/kubevirt"'

    Note: Requires ripgrep/rg to work.
EOF
}

main "$@"
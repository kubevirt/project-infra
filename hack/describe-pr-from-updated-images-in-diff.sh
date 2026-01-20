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

set -euo pipefail

PROJECT_INFRA_ROOT=$(readlink --canonicalize "$( dirname "${BASH_SOURCE[0]}" )/..")

function main() {
    headline="Bump prow-deploy images"
    if [ $# -gt 0 ]; then
        headline="$1"
    fi

    cat << EOF
${headline}

FYI @kubevirt/prow-job-taskforce
/cc none

Images updated:
EOF
    for image in $(
        git diff -- "${PROJECT_INFRA_ROOT}/github/ci/prow-deploy" | \
            grep -E '^\+\s+-? image: ' | \
            grep -oE 'quay.io/kubevirtci/[^: @]+' | \
            sort -u
    ); do
        echo "* ${image}"
    done
}

main "$@"

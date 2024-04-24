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
# Copyright the KubeVirt Authors
#
#

set -euo pipefail
set -x

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_INFRA_ROOT=$(readlink --canonicalize ${BASEDIR}/..)

IMAGES=( botreview phased referee rehearse release-blocker )
for image_dir in "${IMAGES[@]}"; do
    image_name="quay.io/kubevirtci/${image_dir/#\.\//}"
    if ! "${PROJECT_INFRA_ROOT}/hack/update-deployments-with-latest-image.sh"  "${image_name}"; then
        echo "Failed to update prow deployments using image $image_name"
        exit 1
    fi
    echo "Updated prow deployments using image $image_name"
done

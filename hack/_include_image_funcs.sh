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

# _include_image_funcs.sh contains bash functions that are being used in multiple
# places

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_INFRA_ROOT=$(readlink --canonicalize ${BASEDIR}/..)

function kubevirtci_images_used_in_manifests() {
    find "${PROJECT_INFRA_ROOT}/github/ci/prow-deploy" -name '*.yaml' -print0 | \
        xargs -0 grep -hoE 'quay.io/kubevirtci/[^: @]+' | \
        sort -u | \
        cut -d'/' -f3
}

# Returns the latest tag for a given image from quay.io
function latest_image_tag() {
    local image_name="$1"
    local latest_tag
    latest_tag=$(curl -s "https://quay.io/api/v1/repository/${image_name#quay.io/}/tag/?limit=1" | \
        jq -r '.tags[] | select(.name != "latest") | .name' | head -n1)

    if [[ -z "$latest_tag" ]]; then
        echo "[WARN] Could not determine latest tag for $image_name" >&2
        return 1
    fi

    echo "$latest_tag"
}

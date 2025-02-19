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

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
PROJECT_INFRA_ROOT=$(readlink --canonicalize "${BASEDIR}/..")

source "${BASEDIR}/_include_image_funcs.sh"

# Find all Containerfiles / Dockerfiles
function containerfiles() {
    git ls-files | grep -E '(Containerfile|Dockerfile)$'
}

# Extract all FROM image references from a file
function images_from_containerfile() {
    awk '/^FROM /{
        for (i = 2; i <= NF; i++) {
            if (toupper($i) == "AS") break
            printf "%s%s", $i, (i < NF && toupper($(i+1)) != "AS" ? " " : "\n")
        }
    }' "$1"
}

function main() {
    for file in $(containerfiles); do
        echo "[INFO] Processing ${file}"

        for image in $(images_from_containerfile "${file}"); do
            repo="${image%:*}"      # e.g., quay.io/kubevirtci/golang
            old_tag="${image##*:}"  # e.g., v20250930-85e32e0

            latest_tag=$(latest_image_tag "${repo}" || true)

            if [[ -z "${latest_tag}" ]]; then
                echo "[WARN] No latest tag found for ${repo}, skipping"
                continue
            fi

            if [[ "${old_tag}" == "${latest_tag}" ]]; then
                echo "[INFO] ${repo} already at latest tag ${old_tag}"
                continue
            fi

            echo "[INFO] Updating ${repo}:${old_tag} → ${latest_tag} in ${file}"
            sed -i "/^FROM /s#${repo}:${old_tag}#${repo}:${latest_tag}#g" "${file}"
        done

        echo "[OK] Finished ${file}"
    done
}

main "$@"

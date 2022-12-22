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

set -euo pipefail
set -x

if ! command -V skopeo; then
    echo "skopeo required!"
    exit 1
fi

function usage() {
    cat <<EOF
usage: $0 [<container-image> [<directory>]]

    Updates all container files inside directory (defaults to 'images/') to use the latest image version

    Requires skopeo
EOF
}

IMAGE_NAME="quay.io/kubevirtci/bootstrap"
if [ $# -gt 1 ]; then
    if [ ! -d "$2" ]; then
        usage
        echo "$2 is not a directory!"
        exit 1
    fi
    image_dir="$2"
else
    if [ $# -gt 0 ]; then
        if [[ $1 == "-h" ]] || [[ $1 == "--help" ]]; then
            usage
            exit 0
        fi
        IMAGE_NAME="$1"
    fi
    image_dir="$(readlink --canonicalize "$(cd "$(cd "$(dirname "$0")" && pwd)"'/../images' && pwd)")"
fi

latest_image_tag=$(skopeo list-tags "docker://$IMAGE_NAME" | jq -r '.Tags[] | select( match("^v[0-9]+-[a-z0-9]{7,8}$") )' | tail -1)
if [ -z "$latest_image_tag" ]; then
    echo "Couldn't find latest_image_tag"
    exit 1
fi
IMAGE_NAME_WITH_TAG="$IMAGE_NAME:$latest_image_tag"

replace_regex='s#'"$IMAGE_NAME"'(@sha256\:[a-z0-9]+|:v[0-9]+-[a-z0-9]{7,8})#'"$IMAGE_NAME_WITH_TAG"'#g'

find "$image_dir" -regextype egrep -regex '.*(Docker|Container)file$' -exec sed -i -E "$replace_regex" {} +

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
usage: $0 <docker-image> [<directory>]

    Updates all docker files inside directory (defaults to 'images/') to use the latest image version

    Requires skopeo
EOF
}

if [ $# -lt 1 ]; then
    usage
    exit 1
fi

if [ $# -gt 1 ]; then
    if [ ! -d "$2" ]; then
        echo "$2 is not a directory!"
        exit 1
    fi
    docker_file_dir="$2"
else
    docker_file_dir="$(readlink --canonicalize "$(cd "$(cd "$(dirname "$0")" && pwd)"'/../images' && pwd)")"
fi

IMAGE_NAME="$1"
latest_image_tag=$(skopeo list-tags "docker://$IMAGE_NAME" | jq -r '.Tags[] | select( match("v[0-9]+-[a-z0-9]+") )' | tail -1)
if [ -z "$latest_image_tag" ]; then
    echo "Couldn't find latest_image_tag"
    exit 1
fi
IMAGE_NAME_WITH_TAG="$IMAGE_NAME:$latest_image_tag"

replace_regex='s#^FROM '"$IMAGE_NAME"'(@sha256\:|:v?[a-z0-9]+-).*$#FROM '"$IMAGE_NAME_WITH_TAG"'#g'

find "$docker_file_dir" -name 'Dockerfile' -exec sed -i -E "$replace_regex" {} +

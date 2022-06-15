#!/bin/bash
set -euo pipefail
set -x

if ! command -V skopeo; then
    echo "skopeo required!"
    exit 1
fi

job_dir="$(readlink --canonicalize "$(cd "$(cd "$(dirname "$0")" && pwd)" && pwd)")"

IMAGE_NAME="$1"
latest_image_tag=$(skopeo list-tags "docker://$IMAGE_NAME" | jq -r '.Tags[] | select( match("v[0-9]+-[a-z0-9]+") )' | tail -1)
if [ -z "$latest_image_tag" ]; then
    echo "Couldn't find latest_image_tag"
    exit 1
fi
IMAGE_NAME_WITH_TAG="$IMAGE_NAME:$latest_image_tag"

replace_regex='s#'"$IMAGE_NAME"'(@sha256\:[a-z0-9]+|:v?[a-z0-9-]+)#'"$IMAGE_NAME_WITH_TAG"'#g'

find -P "$job_dir" -type f -links 1 -regextype egrep -regex '.*\.sh' -exec sed -i -E "$replace_regex" {} +

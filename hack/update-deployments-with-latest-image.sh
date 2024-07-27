#!/bin/bash
set -euo pipefail
set -x

if ! command -V skopeo; then
    echo "skopeo required!"
    exit 1
fi

if [ $# -gt 1 ]; then
    image_tag="$2"
fi
deployment_dir="$(readlink --canonicalize "$(cd "$(cd "$(dirname "$0")" && pwd)"'/../github/ci/prow-deploy/kustom/base/manifests/local' && pwd)")"

IMAGE_NAME="$1"
if [ -z "$image_tag" ]; then
    latest_image_tag=$(skopeo list-tags "docker://$IMAGE_NAME" | jq -r '.Tags[] | select( match("^v?[0-9]+-[a-z0-9]{7,9}$") )' | tail -1)
    if [ -z "$latest_image_tag" ]; then
        echo "Couldn't find latest_image_tag"
        exit 1
    fi
    IMAGE_NAME_WITH_TAG="$IMAGE_NAME:$latest_image_tag"
else
    IMAGE_NAME_WITH_TAG="$IMAGE_NAME:$image_tag"
fi

replace_regex='s#'"$IMAGE_NAME"'(@sha256\:|:v?[a-z0-9]+-).*$#'"$IMAGE_NAME_WITH_TAG"'#g'

find "$deployment_dir" -regextype egrep -regex '.*-deployment\.yaml' -exec sed -i -E "$replace_regex" {} +

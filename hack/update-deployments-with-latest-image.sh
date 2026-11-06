#!/bin/bash
set -euo pipefail
set -x

if ! command -V skopeo; then
    echo "skopeo required!"
    exit 1
fi

image_tag=""
if [ $# -gt 1 ]; then
    image_tag="$2"
fi

IMAGE_NAME="$1"
project_infra_root="$(cd "$(dirname "$0")/.." && pwd)"
if [ -z "$image_tag" ]; then
    image_tag=$(
        cd "$project_infra_root"
        go run ./robots/image-bumper latest-tag "$IMAGE_NAME"
    )
    if [ -z "$image_tag" ]; then
        echo "Couldn't find latest_image_tag"
        exit 1
    fi
fi
IMAGE_NAME_WITH_TAG="$IMAGE_NAME:$image_tag"

declare -a deployment_dirs=('github/ci/prow-deploy/kustom/base/manifests/local' 'github/ci/prow-deploy/kustom/overlays/prow-workloads/resources')

for i in "${deployment_dirs[@]}";
do
    deployment_dir="${project_infra_root}/${i}"

    replace_regex='s#'"$IMAGE_NAME"'(@sha256\:|:v?[a-z0-9]+-).*$#'"$IMAGE_NAME_WITH_TAG"'#g'

    find "$deployment_dir" -regextype egrep -regex '.*.yaml' -exec sed -i -E "$replace_regex" {} +
done

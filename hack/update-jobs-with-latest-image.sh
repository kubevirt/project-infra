#!/bin/bash
set -euo pipefail
set -x

if ! command -V skopeo; then
    echo "skopeo required!"
    exit 1
fi

if [ $# -gt 1 ]; then
    if [ ! -d "$2" ]; then
        echo "$2 is not a directory!"
        exit 1
    fi
    job_dir="$2"
else
    job_dir="$(readlink --canonicalize "$(cd "$(cd "$(dirname "$0")" && pwd)"'/../github/ci/prow-deploy/files/jobs' && pwd)")"
fi

IMAGE_NAME="$1"
project_infra_root="$(cd "$(dirname "$0")/.." && pwd)"
latest_image_tag=$(
    cd "$project_infra_root"
    go run ./robots/image-bumper latest-tag "$IMAGE_NAME"
)
if [ -z "$latest_image_tag" ]; then
    echo "Couldn't find latest_image_tag"
    exit 1
fi
IMAGE_NAME_WITH_TAG="$IMAGE_NAME:$latest_image_tag"

replace_regex='s#'"$IMAGE_NAME"'(@sha256\:|:v?[a-z0-9]+-).*$#'"$IMAGE_NAME_WITH_TAG"'#g'

find "$job_dir" -regextype egrep -regex '.*-(periodics|presubmits|postsubmits)(-master|-main)?\.yaml' -exec sed -i -E "$replace_regex" {} +

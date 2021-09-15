#!/bin/bash
set -euo pipefail
set -x

if ! command -V skopeo; then
    echo "skopeo required!"
    exit 1
fi

IMAGE_NAME="$1"
latest_image_tag=$(skopeo list-tags "docker://$IMAGE_NAME" | jq -r '.Tags[]' | sort -rV | head -1)
IMAGE_NAME_WITH_TAG="$IMAGE_NAME:$latest_image_tag"

replace_regex='s#'"$IMAGE_NAME"'(@sha256\:|:v[a-z0-9]+-).*$#'"$IMAGE_NAME_WITH_TAG"'#g'

job_dir="$(readlink --canonicalize "$(cd "$(cd "$(dirname $0)" && pwd)"'/../github/ci/prow-deploy/files/jobs' && pwd)")"

find "$job_dir" -regextype egrep -regex '.*-(periodics|presubmits|postsubmits)\.yaml' -exec sed -i -E $replace_regex {} +

#!/usr/bin/env bash
set -xeuo pipefail

image_name="$1"

get_image_tag() {
    local current_commit today
    current_commit="$(git rev-parse HEAD)"
    today="$(date +%Y%m%d)"
    echo "v${today}-${current_commit:0:7}"
}

image_tag="$(get_image_tag)"
bazel run --define container_tag="${image_tag}" "//robots/cmd/${image_name}:push"
docker pull "quay.io/kubevirtci/${image_name}:${image_tag}"
docker tag "quay.io/kubevirtci/${image_name}:${image_tag}" "quay.io/kubevirtci/${image_name}:latest"
docker push "quay.io/kubevirtci/${image_name}:latest"
bash -x ../../../hack/update-jobs-with-latest-image.sh "quay.io/kubevirtci/${image_name}"

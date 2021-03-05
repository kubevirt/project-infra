#!/usr/bin/env bash
set -euo pipefail

get_image_tag() {
    local current_commit today
    current_commit="$(git rev-parse HEAD)"
    today="$(date +%Y%m%d)"
    echo "v${today}-${current_commit:0:7}"
}

bazel run --define container_tag="$(get_image_tag)" //robots/cmd/flakefinder:push
bash -x ../../../hack/update-jobs-with-latest-image.sh quay.io/kubevirtci/flakefinder

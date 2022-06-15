#!/usr/bin/env bash
set -xeuo pipefail

script_dirname=$(cd "$(dirname $0)" && pwd)
source "$script_dirname/../../hack/print-workspace-status.sh"

image_name="$1"

bazel run "//robots/cmd/${image_name}:push"
docker pull "quay.io/kubevirtci/${image_name}:${docker_tag}"
short_docker_tag="v$(date -u '+%Y%m%d')-$(git show -s --format=%h)"
docker tag "quay.io/kubevirtci/${image_name}:${docker_tag}" "quay.io/kubevirtci/${image_name}:${short_docker_tag}"
docker push "quay.io/kubevirtci/${image_name}:${short_docker_tag}"
docker tag "quay.io/kubevirtci/${image_name}:${docker_tag}" "quay.io/kubevirtci/${image_name}:latest"
docker push "quay.io/kubevirtci/${image_name}:latest"
bash -x "$script_dirname/../../hack/update-jobs-with-latest-image.sh" "quay.io/kubevirtci/${image_name}"
bash -x "$script_dirname/../../hack/update-scripts-with-latest-image.sh" "quay.io/kubevirtci/${image_name}"

#!/bin/bash
set -euo pipefail

IMAGE_NAME="$1"

job_dir="$(cd "$(cd "$(dirname $0)" && pwd)"'/../github/ci/prow/files/jobs' && pwd)"

docker pull "$IMAGE_NAME"
sha_id=$(docker images --digests "$IMAGE_NAME" | grep 'latest ' | head -1 | awk '{ print $3 }')

# shellcheck disable=SC2086
for file in $(grep -l 'image: .*'"$IMAGE_NAME" $job_dir/**/**/*-periodics.yaml | sort | uniq); do
    sed -i -E 's#'"$IMAGE_NAME"'@sha256\:[a-z0-9]+#'"$IMAGE_NAME"'@'"$sha_id"'#g' \
        "$file"
done

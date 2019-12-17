#!/bin/bash
set -euo pipefail

docker pull kubevirtci/flakefinder
sha_id=$(docker images --digests kubevirtci/flakefinder | grep 'latest ' | awk '{ print $3 }')

for file in $(grep -l 'image: .*/kubevirtci/flakefinder' ../../github/ci/prow/files/jobs/**/*-periodics.yaml); do
    sed -i -E 's/index\.docker\.io\/kubevirtci\/flakefinder@sha256\:[a-z0-9]+/'"index\.docker\.io\/kubevirtci\/flakefinder@$sha_id"'/g' \
        "$file"
done

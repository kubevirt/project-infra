#!/bin/bash
set -euo pipefail

docker pull kubevirtci/indexpagecreator
sha_id=$(docker images --digests kubevirtci/indexpagecreator | grep 'latest ' | head -1 | awk '{ print $3 }')

for file in $(grep -l 'image: .*/kubevirtci/indexpagecreator' ../../github/ci/prow/files/jobs/**/**/*-periodics.yaml); do
    sed -i -E 's#index.docker.io/kubevirtci/indexpagecreator@sha256:[a-z0-9]+#index.docker.io/kubevirtci/indexpagecreator@'"$sha_id"'#g' \
        "$file"
done

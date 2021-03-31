#!/bin/bash
set -euo pipefail

docker pull quay.io/kubevirtci/indexpagecreator
sha_id=$(docker images --digests quay.io/kubevirtci/indexpagecreator | grep 'latest ' | head -1 | awk '{ print $3 }')

for file in $(grep -l 'image: .*/kubevirtci/indexpagecreator' ../../github/ci/prow/files/jobs/**/**/*-periodics.yaml); do
    sed -i -E 's#quay.io/kubevirtci/indexpagecreator@sha256:[a-z0-9]+#quay.io/kubevirtci/indexpagecreator@'"$sha_id"'#g' \
        "$file"
done

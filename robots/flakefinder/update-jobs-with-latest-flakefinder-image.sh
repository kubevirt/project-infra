#!/bin/bash
set -euo pipefail

docker pull kubevirtci/flakefinder
sha_id=$(docker images --digests kubevirtci/flakefinder | grep 'latest ' | awk '{ print $3 }')

sed -i -E 's/index\.docker\.io\/kubevirtci\/flakefinder@sha256\:[a-z0-9]+/'"index\.docker\.io\/kubevirtci\/flakefinder@$sha_id"'/g' \
    ../../github/ci/prow/files/jobs/kubevirt/kubevirt-periodics.yaml

#!/bin/bash
set -euo pipefail

docker pull kubevirtci/release-blocker
sha_id=$(docker images --digests kubevirtci/release-blocker | grep 'latest ' | head -1 | awk '{ print $3 }')
file="../../github/ci/prow/templates/release-blocker_deployment.yaml"

sed -i -E "s|image\:.*release-blocker.*|image\: index\.docker\.io\/kubevirtci\/release-blocker@$sha_id|g" "$file"

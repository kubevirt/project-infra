#!/bin/bash
set -euo pipefail

IMAGE_NAME="$1"

job_dir="$(cd "$(cd "$(dirname $0)" && pwd)"'/../github/ci/prow/files/jobs' && pwd)"

docker pull "$IMAGE_NAME"
sha_id=$(docker images --digests "$IMAGE_NAME" | grep 'latest ' | head -1 | awk '{ print $3 }')

command -V skopeo && image_tag=$(skopeo inspect "docker://$IMAGE_NAME@$sha_id" | jq -r ' [ .RepoTags[] | select( test( "latest" ) != true ) ] | sort | .[-1] ' )

replace_regex='s#'"$IMAGE_NAME"'(@sha256\:[a-z0-9]+|:v[a-z0-9]+-[a-z0-9]+)#'"$IMAGE_NAME"
if [ -n "$image_tag" ]; then
    replace_regex+=':'"$image_tag"'#g'
else
    replace_regex+='@'"$sha_id"'#g'
fi

find "$job_dir" -regextype egrep -regex '.*-(periodics|presubmits|postsubmits)\.yaml' -exec sed -i -E $replace_regex {} +

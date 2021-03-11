#!/bin/bash

git_commit="$(git describe --tags --always --dirty)"
build_date="$(date -u '+%Y%m%d')"
docker_tag="v${build_date}-${git_commit}"

cat <<EOF
DOCKER_TAG ${docker_tag}
EOF

#!/bin/bash
CWD=$(pwd)
SCRIPT_DIR=$(cd $(dirname $0) && pwd)
set -euo pipefail

cat $DOCKER_PASSWORD | docker login --username $(cat $DOCKER_USER) --password-stdin

WORK_DIR_PARENT="/tmp/github.com/slintes"
mkdir -p "$WORK_DIR_PARENT"
cd "$WORK_DIR_PARENT"
WORK_DIR="$WORK_DIR_PARENT/docker-nfs-ganesha"
[ ! -d "$WORK_DIR" ] && git clone https://github.com/slintes/docker-nfs-ganesha.git

cd "$WORK_DIR" || exit 1

IMAGE_NAME="kubevirtci/nfs-ganesha"

docker build "$(pwd)" -t "$IMAGE_NAME"
docker push "$IMAGE_NAME:latest"
sha_id=$(docker images --digests "$IMAGE_NAME" | grep 'latest ' | awk '{ print $3 }')
echo "Pushed nfs-ganesha as image $IMAGE_NAME with digest $sha_id"

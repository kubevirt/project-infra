#!/bin/bash
CWD=$(pwd)
SCRIPT_DIR=$(cd $(dirname $0) && pwd)
set -euo pipefail

cat $QUAY_PASSWORD | docker login quay.io --username $(cat $QUAY_USER) --password-stdin

export GIMME_GO_VERSION="1.13"
eval $(gimme)

WORK_DIR_PARENT="/tmp/github.com/openshift"
mkdir -p "$WORK_DIR_PARENT"
cd "$WORK_DIR_PARENT"
WORK_DIR="$WORK_DIR_PARENT/ci-tools"
[ ! -d "$WORK_DIR" ] && git clone https://github.com/dhiller/ci-tools.git

cd "$WORK_DIR"
git checkout add-signoff-to-autoowners-2
git pull
export GOPROXY=off
export GOFLAGS=-mod=vendor
make install

IMAGE_NAME="quay.io/kubevirtci/autoowners"

cd "$SCRIPT_DIR"
eval $(go env|grep GOPATH)
cp "$GOPATH/bin/autoowners" .
docker build "$SCRIPT_DIR" -t "$IMAGE_NAME"
docker push "$IMAGE_NAME:latest"
sha_id=$(docker images --digests "$IMAGE_NAME" | grep 'latest ' | awk '{ print $3 }')
echo "Pushed autoowners as image $IMAGE_NAME with digest $sha_id"

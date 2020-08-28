#!/usr/bin/env bash
TEST_INFRA_DIR=/tmp/test-infra

pushd /tmp/test-infra
VERSION=$(git rev-parse HEAD)
BASE_DIR=base/manifests/test_infra
VERSION_DIR=$BASE_DIR/$VERSION
popd

if [[ -d ${VERSION_DIR} ]]; then
    echo "Manifests already at latest version"
    exit 1
fi

mkdir $VERSION_DIR
pushd $BASE_DIR
rm current
ln -s $VERSION current
cp -a $TEST_INFRA_DIR/config/prow/cluster/* $VERSION

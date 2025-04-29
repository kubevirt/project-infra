#!/bin/bash

set -x
set -euo pipefail

function usage {
    cat <<EOF
usage: $0 <Containerfile/Dockerfile>

    Bump a fedora based container to the latest available fedora version

EOF
}


if [ ! "$#" -eq 1 ]; then
    usage
    exit 1
fi

BUILD_FILE=$1

if [ ! -f "$BUILD_FILE" ];  then
    echo "Containerfile not found at $BUILD_FILE"
    exit 1
fi

# Get latest version of Fedora available

LATEST_FEDORA=$(curl -s -L https://fedoraproject.org/releases.json | jq -r '[.[].version|select(test("^[0-9]+$"))]|max')

sed -i s"/fedora:[0-9][0-9]/fedora:$LATEST_FEDORA/"  $BUILD_FILE

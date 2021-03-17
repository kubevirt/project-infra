#!/usr/bin/env bash
set -xeuo pipefail

script_dirname=$(cd "$(dirname $0)" && pwd)
source "$script_dirname/../../hack/print-workspace-status.sh"

if [[ "${docker_tag}" =~ dirty ]]; then
    echo "Build is dirty"
    exit 1
fi

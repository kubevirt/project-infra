#!/usr/bin/env bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright the KubeVirt Authors.
#
#

set -euo pipefail

function usage() {
    cat <<EOF
usage: $0 <gh-token-path> <arg1> [... <argn>]
       $0 <gh-token-path> --help
EOF
}

function main() {
    if [ $# -lt 1 ] || [ ! -f $1 ]; then
        usage
        exit 1
    fi

    oauth_token_dir="$(realpath $(dirname $1))"
    oauth_file_name="$(basename $1)"
    shift

    podman run \
        -v "$oauth_token_dir:/etc/tokens:z" \
        --rm quay.io/kubevirtci/migratestatus:v20250326-66cd380 \
        --github-token-path "/etc/tokens/$oauth_file_name" $@
}

main "$@"

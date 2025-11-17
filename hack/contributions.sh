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
usage: $0 <path-to-oauth-token> [<arg-1>...<arg-n>]
       $0 --help

       Example: $0 /path/to/github/oauth --username dhiller --repo project-infra --months 12
EOF
}

if [[ $# -lt 1 ]]; then
    usage
    exit 1
fi

if [[ "$1" == "--help" ]]; then
    podman run \
        --rm \
        quay.io/kubevirtci/contributions:v20250417-cd6921f \
        --help
    exit 1
fi

oauth_token="$1"
if [[ ! -f "${oauth_token}" ]]; then
    usage
    exit 1
fi
shift

project_infra_dir="$(realpath "$(dirname "${BASH_SOURCE[0]}")/..")"

temp_dir=$(mktemp -d /tmp/contributions-XXXXXX)

podman run \
    -v "${temp_dir}:/tmp:z" \
    -v "$(dirname "${oauth_token}"):/etc/github:Z" \
    -v "${project_infra_dir}:/project-infra:z" \
    --rm \
    quay.io/kubevirtci/contributions:v20250417-cd6921f \
    --github-token "/etc/github/$(basename "${oauth_token}")" \
    --orgs-file-path "/project-infra/github/ci/prow-deploy/kustom/base/configs/current/orgs/orgs.yaml" \
    "$@" 2>&1 | tee "${temp_dir}/output.txt"
report_file=$(grep -oE '[^/]*\.yaml' "${temp_dir}/output.txt" | head -n1)

echo "Detailed user activity report: ${temp_dir}/${report_file}"

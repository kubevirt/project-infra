#!/bin/bash

# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright the KubeVirt Authors.

set -eu -o pipefail

# Process command line arguments
PROGNAME='mirror-crio'

# Constants and default values
BUCKET_NAME='kubevirtci-crio-mirror'
CRIO_VERSIONS=

usage() {
    cat <<EOF
Helper script for mirroring CRI-O repositories to a cloud storage bucket.

Usage:
  ${PROGNAME} [options]

Options:
  --bucket=<bucket-name>
    Name of the bucket to which the CRI-O repositories will be mirrored.
    (Default: ${BUCKET_NAME:-unset})

  --crio-versions=<version1,..,versionN>
    Comma-separated list of CRI-O versions which must be mirrored.
    (Default: ${CRIO_VERSIONS:-unset})
EOF
}

die(){ usage >&2; exit 64; }  # EX_USAGE

# Parse command line arguments
CMDLINE=$(
    opts='bucket:,crio-versions:,help'

    getopt \
        --name="${PROGNAME}" \
        --longoptions="${opts}" \
        --options='h' \
        -- "$@"
) || die

eval set -- "${CMDLINE}"

# Convert command line arguments to local variables
while [ "$#" -gt 0 ]; do
    case $1 in
        --bucket) BUCKET_NAME=$2; shift;;
        # IFS trick converts a comma separated list to a bash array
        --crio-versions) IFS=',' CRIO_VERSIONS=($2); shift;;

        --help|-H) usage;  exit;;
        --)        shift; break;;  # End of options
        *)         die;;           # Should not happen
    esac
    shift
done

mirror_crio_repo_for_version()
{
    local channel repo_id repo_url version

    # https://cri-o.io/#distribution-packaging
    version="$1"
    channel="stable"
    repo_id="isv_cri-o_${channel}_v${version}"
    repo_url="https://download.opensuse.org/repositories/isv:/cri-o:/${channel}:/v${version}/rpm/isv:cri-o:${channel}:v${version}.repo"

    echo "[INFO] Mirroring CRI-O v${version} repository" >&2
    curl -sSLR --fail-with-body --retry 2 "${repo_url}" -o "${repo_id}".repo
    dnf reposync --config="${repo_id}.repo" --repo="${repo_id}" \
        --destdir="./crio-mirror" --download-metadata \
        --newest-only --remote-time
}

workdir=$(mktemp -d)
trap 'rm -rf "${workdir}"' EXIT

cd "${workdir}"

for version in "${CRIO_VERSIONS[@]}"; do
    mirror_crio_repo_for_version "${version}"
done

gcloud storage rsync ./crio-mirror gs://"${BUCKET_NAME}" \
    --recursive \
    --delete-unmatched-destination-objects

cd - >/dev/null

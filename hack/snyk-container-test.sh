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
usage: $0 [{base-image-name}]

       checks one or more image(s) for vulnerabilities with Snyk

       Assumptions:
       * all images are located at quay.io/kubevirtci/{base-image-name} .
       * all base image names are names of subfolders below ./images

       if {base-image-name} is given, it will check the image defined by the Containerfile in the subfolder.
       Otherwise it will automatically check images based on names of all subfolders below './images/'.
EOF
}

function check_binary_in_path() {
    if ! command -v "$1"; then
        echo "$1 binary not found in executable path"
        exit 1
    fi
}

function check_container() {
    container_file="$1"
    if [ ! -f "${container_file}" ]; then
        echo "${container_file} does not exist"
        return
    fi
    image_name="$(echo "${container_file}" | cut -d '/' -f 3)"
    full_image_name="quay.io/kubevirtci/${image_name}"
    latest_image_tag=$(skopeo list-tags "docker://${full_image_name}" | jq -r '.Tags[] | select( match("^v?[0-9]+-[a-z0-9]{7,9}$") )' | tail -1)
    output=$(
        snyk container test "${full_image_name}:${latest_image_tag}" \
            --exclude-base-image-vulns \
            --file="${container_file}"
    )
    return_code=$?
    case $return_code in
    0) ;;

    *)
        # 1: action_needed (scan completed), vulnerabilities found
        # 2: failure, try to re-run command. Use -d to output the debug logs.
        # 3: failure, no supported projects detected
        echo "${output}"
        exit $return_code
        ;;
    esac
}

function main() {
    if [ $# -gt 0 ] && [[ "$1" =~ ^(-h|--help)$ ]]; then
        usage
        exit 0
    fi

    check_binary_in_path snyk
    check_binary_in_path jq

    if [ $# -ne 0 ]; then
        container_file="./images/$1/Containerfile"
        check_container "${container_file}"
    else
        while IFS= read -r -d '' container_file; do
            check_container "${container_file}"
        done < <(find ./images/ -name Containerfile -type f -print0)
    fi
}

main "$@"

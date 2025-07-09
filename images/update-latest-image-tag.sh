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

set -x
set -e

function main() {
    while getopts "h" opt; do
        case "$opt" in
            h)
                help
                exit 0
                ;;
            *)
                echo "Invalid argument: $opt"
                help
                exit 1
        esac
    done
    shift $((OPTIND-1))
    local build_target="${1:?}"
    local registry="${2:?}"
    local registry_org="${3:?}"
    local full_image_name

    full_image_name="$(
        get_full_image_name \
            "$registry" \
            "$registry_org" \
            "${build_target##*/}" \
    )"

    update_latest_image_tag "$full_image_name"
}

function help() {
    cat <<EOF
    Usage:
        $0 [OPTIONS] BUILD_TARGET REGISTRY REGISTRY_ORG

    Update `latest` tag for published infra images.

    OPTIONS
        -h  Show this help message and exit.
EOF
}

update_latest_image_tag() {
    local full_image_name="${1:?}"

    local latest_image_tag
    latest_image_tag=$( skopeo list-tags "docker://${full_image_name}" | jq -r '.Tags[] | select(. | startswith("v"))' | sort | tail -n 1 )

    podman pull "${full_image_name}:${latest_image_tag}"
    podman image tag "${full_image_name}:${latest_image_tag}" "${full_image_name}:latest"
    podman push "${full_image_name}:latest"
}

get_full_image_name() {
    local registry="${1:?}"
    local registry_org="${2:?}"
    local image_name="${3:?}"

    echo "${registry}/${registry_org}/${image_name}"
}

main "$@"

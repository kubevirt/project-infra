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

set -eux -o pipefail

archs=(amd64 arm64 s390x)

main() {
    local build_only=''
    local project_infra_dir
    project_infra_dir="$(readlink --canonicalize "$(pwd)/../")"
    while getopts "abp:h" opt; do
        case "$opt" in
	    a)
		archs=(amd64)
		;;
            b)
                build_only=true
                ;;
            p)  project_infra_dir="$OPTARG"
                ;;
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
    local full_image_name image_tag

    image_tag="$(get_image_tag)"
    full_image_name="$(
        get_full_image_name \
            "$registry" \
            "$registry_org" \
            "${build_target##*/}" \
            "$image_tag"
    )"

    (
        cd "$build_target"

        build_image "$build_target" "$full_image_name" "$project_infra_dir"
    )
    [[ $build_only ]] && return
    publish_image "$full_image_name"
    publish_manifest "$full_image_name"
}

help() {
    cat <<EOF
    Usage:
        ./publish_multiarch_image.sh [OPTIONS] BUILD_TARGET REGISTRY REGISTRY_ORG

    Build and publish multiarch infra images.

    OPTIONS
        -a  Build only amd64 image
        -h  Show this help message and exit.
        -b  Only build the image and exit. Do not publish the built image.
EOF
}

get_image_tag() {
    local current_commit today
    current_commit="$(git rev-parse HEAD)"
    today="$(date +%Y%m%d)"
    echo "v${today}-${current_commit:0:7}"
}

build_image() {
    local build_target="${1:?}"
    local image_name="${2:?}"
    local project_infra_dir="${3:-}"
    build_args=()
    if [ -d "$project_infra_dir" ]; then
        build_args=( "-v" "$project_infra_dir:/project-infra:z" )
    fi
    # add qemu-user-static
    # - https://github.com/linuxserver/docker-qemu-static
    podman run --rm --privileged quay.io/linuxserver.io/qemu-static --reset -p yes
    # build multi-arch images
    for arch in ${archs[*]};do
        podman build --platform="linux/${arch}" "${build_args[@]}" --build-arg ARCH=${arch} --build-arg IMAGE_ARG=${build_target} . -t "${image_name}-${arch}" -t "${build_target}-${arch}"
    done
}

publish_image() {
    local full_image_name="${1:?}"
    for arch in ${archs[*]};do
        podman push "${full_image_name}-${arch}"
    done
}

publish_manifest() {
    local amend
    local full_image_name="${1:?}"
    amend=""
    for arch in ${archs[*]};do
        amend+=" --amend ${full_image_name}-${arch}"
    done
    podman manifest create ${full_image_name} ${amend}
    podman manifest push ${full_image_name} "docker://${full_image_name}"
}

get_full_image_name() {
    local registry="${1:?}"
    local registry_org="${2:?}"
    local image_name="${3:?}"
    local tag="${4:?}"

    echo "${registry}/${registry_org}/${image_name}:${tag}"
}

main "$@"

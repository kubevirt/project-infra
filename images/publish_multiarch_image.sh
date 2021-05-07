#!/bin/bash -xe
archs=(amd64 arm64)

main() {
    local build_only
    while getopts "bh" opt; do
        case "$opt" in
            b)
                build_only=true
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

        build_image "$build_target" "$full_image_name" "$baseimage"
    )
    [[ $build_only ]] && return
    publish_image "$full_image_name"
    publish_manifest "$full_image_name"
}

help() {
    cat <<EOF
    Usage:
        ./publish_image.sh [OPTIONS] BUILD_TARGET REGISTRY REGISTRY_ORG

    Build and publish infra images.

    OPTIONS
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

get_base_image() {
    local architecture="${1:?}"
    declare -A archimage="$(cat BASEIMAGE)"
    echo ${archimage[$architecture]}
}

build_image() {
    local build_target="${1:?}"
    local image_name="${2:?}"
    # add qemu-user-static
    docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
    # build multi-arch images
    for arch in ${archs[*]};do
        # get baseimage
        local baseimage=$(get_base_image ${arch})
        if [ -f "Dockerfile" ]; then
            docker build --build-arg ARCH=${arch} --build-arg  BASEIMAGE=${baseimage} --build-arg IMAGE_ARG=${build_target} . -t "${image_name}-${arch}" -t "${build_target}-${arch}"
	else
            docker build --build-arg ARCH=${arch} --build-arg  BASEIMAGE=${baseimage} --build-arg IMAGE_ARG=${build_target} -f Dockerfile.${arch} -t "${image_name}-${arch}" -t "${build_target}-${arch}" .
	fi
    done
}

publish_image() {
    local full_image_name="${1:?}"
    for arch in ${archs[*]};do
        docker push "${full_image_name}-${arch}"
    done
}

publish_manifest() {
    export DOCKER_CLI_EXPERIMENTAL="enabled"
    local amend
    local full_image_name="${1:?}"
    amend=""
    for arch in ${archs[*]};do
        amend+=" --amend ${full_image_name}-${arch}"
    done
    docker manifest create ${full_image_name} ${amend}
    docker manifest push --purge ${full_image_name}
}

get_full_image_name() {
    local registry="${1:?}"
    local registry_org="${2:?}"
    local image_name="${3:?}"
    local tag="${4:?}"

    echo "${registry}/${registry_org}/${image_name}:${tag}"
}

main "$@"

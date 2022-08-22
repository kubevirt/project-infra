#!/bin/bash -xe
archs=(amd64 arm64)

main() {
    local build_only local_base_image
    local_base_image=false
    while getopts "ablh" opt; do
        case "$opt" in
	    a)
		archs=(amd64)
		;;
            b)
                build_only=true
                ;;
            l)
                local_base_image=true
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
    local full_image_name image_tag base_image

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
	base_image="$(get_base_image)"

        build_image $local_base_image "$build_target" "$full_image_name" "$base_image"
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
        -l  Use local base image
EOF
}

get_base_image() {
    name="$(cat Dockerfile |grep FROM|awk '{print $2}')"
    echo "${name}"
}

get_image_tag() {
    local current_commit today
    current_commit="$(git rev-parse HEAD)"
    today="$(date +%Y%m%d)"
    echo "v${today}-${current_commit:0:7}"
}

build_image() {
    local local_base_image=${1:?}
    local build_target="${2:?}"
    local image_name="${3:?}"
    local base_image="${4:?}"
    # add qemu-user-static
    docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
    # build multi-arch images
    for arch in ${archs[*]};do
        if [[ $local_base_image == false ]]; then
	    docker pull --platform="linux/${arch}" ${base_image}
        fi
        docker build --platform="linux/${arch}" --build-arg ARCH=${arch} --build-arg IMAGE_ARG=${build_target} . -t "${image_name}-${arch}" -t "${build_target}-${arch}"
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

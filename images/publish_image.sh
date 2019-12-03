#!/bin/bash -xe


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

        build_image "$full_image_name"
    )
    [[ $build_only ]] && return
    publish_image "$full_image_name"
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

build_image() {
    local image_name="${1:?}"
    docker build . -t "$image_name"
}

publish_image() {
    local full_image_name="${1:?}"

    docker push "${full_image_name}"
}

get_full_image_name() {
    local registry="${1:?}"
    local registry_org="${2:?}"
    local image_name="${3:?}"
    local tag="${4:?}"

    echo "${registry}/${registry_org}/${image_name}:${tag}"
}

main "$@"
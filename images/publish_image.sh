#!/bin/bash -xe


main() {
    local build_only
    local no_cache
    local project_infra_dir
    project_infra_dir="$(readlink --canonicalize "$(pwd)/../")"
    while getopts "bcp:h" opt; do
        case "$opt" in
            b)
                build_only=true
                ;;
            c)
                no_cache=true
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

        build_image "$no_cache" "$build_target" "$full_image_name" "$project_infra_dir"
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
        -p  Local project-infra directory
EOF
}

get_image_tag() {
    local current_commit today
    current_commit="$(git rev-parse HEAD)"
    today="$(date +%Y%m%d)"
    echo "v${today}-${current_commit:0:7}"
}

build_image() {
    local no_cache="${1:-}"
    local build_target="${2:?}"
    local image_name="${3:?}"
    local project_infra_dir="${4:-}"
    build_args=""
    if [ -d "$project_infra_dir" ]; then
        build_args="-v $project_infra_dir:/project-infra"
    fi
    if [ "$no_cache" = "true" ]; then
        build_args="$build_args --no-cache"
    fi
    podman build $build_args . -t "$image_name" -t "$build_target"
}

publish_image() {
    local full_image_name="${1:?}"

    podman push "${full_image_name}"
}

get_full_image_name() {
    local registry="${1:?}"
    local registry_org="${2:?}"
    local image_name="${3:?}"
    local tag="${4:?}"

    echo "${registry}/${registry_org}/${image_name}:${tag}"
}

main "$@"

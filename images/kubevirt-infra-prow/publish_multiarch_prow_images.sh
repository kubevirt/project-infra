#!/bin/bash -ex

BAZEL_VERSION=3.0.0
PROW_COMMIT=9f5410055c
registry=quay.io
repo=kubevirtci
archs=(arm64 ppc64le)
binaries=(clonerefs initupload entrypoint sidecar)

main() {
    local build_only
    local tag
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
    tag=$(get_image_tag)
    build_binaries
    build_images
    [[ $build_only ]] && return
    push_images
    push_manifest
}

help() {
    cat <<EOF
    Usage:
        ./publish_multiarch_prow_images.sh [OPTIONS]

    Build and publish multiarch prow utility images.

    OPTIONS
        -h  Show this help message and exit.
        -b  Only build the image and exit. Do not publish the built image.
EOF
}

get_base_image() {
    if [ $1 == "arm64" ]; then
        echo "arm64v8/alpine"
    elif [ $1 == "ppc64le" ] ; then
        echo "ppc64le/alpine"
    else
        echo "unsupport arch"
        exit
    fi
}

get_image_tag() {
    local current_commit today
    current_commit="$(git rev-parse HEAD)"
    today="$(date +%Y%m%d)"
    echo "v${today}-${current_commit:0:7}"
}

build_binaries(){
    # Cleanup Env
    rm -rf /usr/bin/bazel /usr/local/bin/python prow test-infra

    # Install bazel
    apt update && apt install -y curl gnupg git python python3\
        && curl -fsSL https://bazel.build/bazel-release.pub.gpg | gpg --dearmor > bazel.gpg \
        && mv bazel.gpg /etc/apt/trusted.gpg.d/ \
        && echo "deb [arch=amd64] https://storage.googleapis.com/bazel-apt stable jdk1.8" | tee /etc/apt/sources.list.d/bazel.list \
        && apt update && apt install -y bazel-${BAZEL_VERSION} \
        && ln -s /usr/bin/bazel-${BAZEL_VERSION} /usr/bin/bazel \
        && ln -s /usr/bin/python3 /usr/local/bin/python
    # Git clone test-infra source code
    git clone https://github.com/kubernetes/test-infra.git
    cd test-infra && git checkout ${PROW_COMMIT}

    # build utility binary
    mkdir ../prow
    for arch in ${archs[*]};do
        for bin in ${binaries[*]};do
            bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_${arch} //prow/cmd/${bin}
            cp bazel-bin/prow/cmd/${bin}/linux_${arch}_pure_stripped/${bin} ../prow/${bin}-${arch}
        done
    done
    cd ../
}

build_images() {
    # add qemu-user-static
    docker run --rm --privileged multiarch/qemu-user-static --reset -p yes
    # build utility images
    for arch in ${archs[*]};do
        baseimage=$(get_base_image ${arch})
        for bin in ${binaries[*]};do
            docker build --build-arg POD_UTILITY=${bin} --build-arg ARCH=${arch} --build-arg BASEIMAGE=${baseimage} . -t ${registry}/${repo}/${bin}-${arch}:${tag}
        done
    done
}

push_images() {
    for arch in ${archs[*]};do
        for bin in ${binaries[*]};do
            docker push ${registry}/${repo}/${bin}-${arch}:${tag}
        done
    done
}

push_manifest() {
    export DOCKER_CLI_EXPERIMENTAL="enabled"
    local amend
    for bin in ${binaries[*]};do
	amend=""
	for arch in ${archs[*]};do
	    amend+=" --amend ${registry}/${repo}/${bin}-${arch}:${tag}"
	done
        docker manifest create ${registry}/${repo}/${bin}:${tag} ${amend}
        docker manifest push --purge ${registry}/${repo}/${bin}:${tag}
    done
}

main "$@"

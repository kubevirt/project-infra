#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

determine_cri_bin() {
	if podman ps >/dev/null 2>&1; then
		echo podman
	elif docker ps >/dev/null 2>&1; then
		echo docker
	else
		>&2 echo "no working container runtime found. Neither docker nor podman seems to work."
		exit 1
	fi
}

cri_bin=$(determine_cri_bin)
echo "Using ${cri_bin} as container runtime"

${cri_bin} run -it --rm -v "$(pwd)":/src -v "${GITHUB_TOKEN}":"/tmp/github_token" ubuntu:latest /src/hack/performance-benchmarks/publish_graph.sh

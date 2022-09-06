#!/usr/bin/env bash
# Copyright 2018 The Kubernetes Authors.
# Copyright 2021 The KubeVirt Authors.
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

# generic runner script, handles DIND, bazelrc for caching, etc.

# Give bazel a well defined output_user_root directory independent of the user
# used in the image. This allows mounting an emptyDir at this location, instead
# of writing into the container overlay.
mkdir -p /tmp/cache/bazel
echo "startup --output_user_root=/tmp/cache/bazel" >> "${HOME}/.bazelrc"
echo "startup --output_user_root=/tmp/cache/bazel" >> "/etc/bazel.bazelrc"

# Check if the job has opted-in to bazel remote caching and if so generate
# .bazelrc entries pointing to the remote cache
export BAZEL_REMOTE_CACHE_ENABLED=${BAZEL_REMOTE_CACHE_ENABLED:-false}
if [[ "${BAZEL_REMOTE_CACHE_ENABLED}" == "true" ]]; then
    echo "Bazel remote cache is enabled, generating .bazelrcs ..."
    /usr/local/bin/create_bazel_cache_rcs.sh
fi

# setup custom certificates for the container registry mirror
setup_ca(){
    if [ -f "${CA_CERT_FILE}" ]; then
        echo "Adding ${CA_CERT_FILE} as a trusted root CA"
        cp "${CA_CERT_FILE}" /etc/pki/ca-trust/source/anchors/

        update-ca-trust
    fi
}

# runs custom podman data root cleanup binary and debugs remaining resources
cleanup_pinc() {
    if [[ "${PODMAN_IN_CONTAINER_ENABLED:-false}" == "true" ]]; then
        echo "Cleaning up after podman"
        podman ps -aq | xargs -r podman rm -f || true
        kill "$(</var/run/podman.pid)" || true
        wait "$(</var/run/podman.pid)" || true
    fi
}

early_exit_handler() {
    cleanup_pinc
}

# setup certificates before anything gets started
setup_ca

# optionally enable ipv6
export PODMAN_IN_CONTAINER_IPV6_ENABLED=${PODMAN_IN_CONTAINER_IPV6_ENABLED:-true}
if [[ "${PODMAN_IN_CONTAINER_IPV6_ENABLED}" == "true" ]]; then
    echo "Enabling IPV6."
    # enable ipv6
    sysctl net.ipv6.conf.all.disable_ipv6=0
    sysctl net.ipv6.conf.all.forwarding=1
    # enable ipv6 iptables
    modprobe -v ip6table_nat
fi

export PODMAN_IN_CONTAINER_ENABLED=${PODMAN_IN_CONTAINER_ENABLED:-false}
if [[ "${PODMAN_IN_CONTAINER_ENABLED}" == "true" ]]; then
    echo "Podman in Container enabled, initializing in podman compatible mode..."
    PODMAN_SOCKET_PATH=/run/podman
    PODMAN_SOCKET=${PODMAN_SOCKET_PATH}/podman.sock
    (
        export HTTP_PROXY=${CONTAINER_HTTP_PROXY}
        export HTTPS_PROXY=${CONTAINER_HTTPS_PROXY}
        export KIND_EXPERIMENTAL_PROVIDER="podman"

        mkdir -p ${PODMAN_SOCKET_PATH}
      
        podman system service \
                -t 0 \
                unix://${PODMAN_SOCKET} \
                >/var/log/podman.log 2>&1 &
        echo "${!}" > /var/run/podman.pid

        ln -s ${PODMAN_SOCKET} /var/run/docker.sock
        # Set podman short-name-mode to permissive
        sed -i 's/short-name-mode="enforcing"/short-name-mode="permissive"/g' /etc/containers/registries.conf
    )
    # the service can be started but the socket not ready, wait for ready
    WAIT_N=0
    MAX_WAIT=5
    while true; do
        # wait for podman socket to be ready
        curl --unix-socket "${PODMAN_SOCKET}" http://d/v3.0.0/libpod/info >/dev/null 2>&1 && break
        if [[ ${WAIT_N} -lt ${MAX_WAIT} ]]; then
            WAIT_N=$((WAIT_N+1))
            echo "Waiting for podman socket to be ready, sleeping for ${WAIT_N} seconds."
            sleep ${WAIT_N}
        else
            echo "Reached maximum attempts, not waiting any longer..."
	    echo "Docker daemon failed to start successfully"
            exit 1
        fi
    done
    printf '=%.0s' {1..80}; echo
    echo "Done setting up podman in container."
fi

trap early_exit_handler INT TERM

# disable error exit so we can run post-command cleanup
set +o errexit

# add $GOPATH/bin to $PATH
export PATH="${GOPATH}/bin:${PATH}"
mkdir -p "${GOPATH}/bin"
# Authenticate gcloud, allow failures
if [[ -n "${GOOGLE_APPLICATION_CREDENTIALS:-}" ]]; then
  gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}" || true
fi

# Set up Container Registry Auth file
mkdir -p "${HOME}/containers" && echo "{}" > "${HOME}/containers/auth.json"
export REGISTRY_AUTH_FILE="${HOME}/containers/auth.json"
# Bazel push expects credentials to be available at ${HOME}/.docker/config.json
mkdir "${HOME}/.docker" && ln -s "${REGISTRY_AUTH_FILE}" "${HOME}/.docker/config.json"

# Use a reproducible build date based on the most recent git commit timestamp.
SOURCE_DATE_EPOCH=$(git log -1 --pretty=%ct || true)
export SOURCE_DATE_EPOCH

# run setup mixins
for file in $(find /etc/setup.mixin.d/ -maxdepth 1 -name '*.sh' -print -quit); do source $file; done

# actually start bootstrap and the job
set -o xtrace
"$@"
EXIT_VALUE=$?
set +o xtrace

# run teardown mixins
for file in $(find /etc/teardown.mixin.d/ -maxdepth 1 -name '*.sh' -print -quit); do source $file; done

# cleanup after job
if [[ "${PODMAN_IN_CONTAINER_ENABLED}" == "true" ]]; then
    echo "Cleaning up after podman in container."
    printf '=%.0s' {1..80}; echo
    cleanup_pinc
    printf '=%.0s' {1..80}; echo
    echo "Done cleaning up after podman in container."
fi

# preserve exit value from job / bootstrap
exit ${EXIT_VALUE}

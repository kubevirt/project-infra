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

# runs custom docker data root cleanup binary and debugs remaining resources
cleanup_dind() {
    if [[ "${DOCKER_IN_DOCKER_ENABLED:-false}" == "true" ]]; then
        if [[ "${DOCKER_DEBUG:-false}" == "true" ]]; then
            echo "Copying docker log to ARTIFACTS"
            cp /var/log/dockerd.log $ARTIFACTS
        fi
        echo "Cleaning up after docker"
        docker ps -aq | xargs -r docker rm -f || true
        kill "$(</var/run/docker.pid)" || true
        wait "$(</var/run/docker.pid)" || true
    fi
}

early_exit_handler() {
    cleanup_dind
}

# setup certificates before anything gets started
setup_ca

# optionally enable ipv6 docker
export DOCKER_IN_DOCKER_IPV6_ENABLED=${DOCKER_IN_DOCKER_IPV6_ENABLED:-false}
if [[ "${DOCKER_IN_DOCKER_IPV6_ENABLED}" == "true" ]]; then
    echo "Enabling IPV6 for Docker."
    # configure the daemon with ipv6
    mkdir -p /etc/docker/
    cat <<EOF >/etc/docker/daemon.json
{
  "ipv6": true,
  "fixed-cidr-v6": "fc00:db8:1::/64"
}
EOF
    # enable ipv6
    sysctl net.ipv6.conf.all.disable_ipv6=0
    sysctl net.ipv6.conf.all.forwarding=1
    # enable ipv6 iptables
    modprobe -v ip6table_nat
fi

# Check if the job has opted-in to docker-in-docker availability.
export DOCKER_IN_DOCKER_ENABLED=${DOCKER_IN_DOCKER_ENABLED:-false}
if [[ "${DOCKER_IN_DOCKER_ENABLED}" == "true" ]]; then
    echo "Docker in Docker enabled, initializing..."

    export DOCKER_DEBUG=${DOCKER_DEBUG:-false}
    if [[ "${DOCKER_DEBUG}" == "true" ]]; then
        mkdir -p /etc/docker/
        # TODO: do not rely on this file not existing!
        cat <<EOF >/etc/docker/daemon.json
{
  "debug": true,
  "log-level": "debug"
}
EOF
    fi

    printf '=%.0s' {1..80}; echo
    # If we have opted in to docker in docker, start the docker daemon,
    (
        if [ -f "/etc/default/docker" ]; then
            source /etc/default/docker
        fi
        /usr/bin/dockerd \
            -p /var/run/docker.pid \
            --data-root=/docker-graph \
            --init-path /usr/libexec/docker/docker-init \
            --userland-proxy-path /usr/libexec/docker/docker-proxy \
            ${DOCKER_OPTS} \
                >/var/log/dockerd.log 2>&1 &
    )
    # the service can be started but the docker socket not ready, wait for ready
    WAIT_N=0
    MAX_WAIT=5
    while true; do
        # docker ps -q should only work if the daemon is ready
        docker ps -q > /dev/null 2>&1 && break
        if [[ ${WAIT_N} -lt ${MAX_WAIT} ]]; then
            WAIT_N=$((WAIT_N+1))
            echo "Waiting for docker to be ready, sleeping for ${WAIT_N} seconds."
            sleep ${WAIT_N}
        else
            echo "Reached maximum attempts, not waiting any longer..."
	    echo "Docker daemon failed to start successfully"
            exit 1
        fi
    done
    printf '=%.0s' {1..80}; echo
    echo "Done setting up docker in docker."
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
if [[ "${DOCKER_IN_DOCKER_ENABLED}" == "true" ]]; then
    echo "Cleaning up after docker in docker."
    printf '=%.0s' {1..80}; echo
    cleanup_dind
    printf '=%.0s' {1..80}; echo
    echo "Done cleaning up after docker in docker."
fi

# preserve exit value from job / bootstrap
exit ${EXIT_VALUE}

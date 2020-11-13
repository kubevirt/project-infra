#!/bin/bash
# Copyright 2020 The KubeVirt Authors.
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

set -eo pipefail

setup_docker_proxy(){
    if [ ! -z "${DOCKER_HTTP_PROXY}" ]; then
        echo "Setting docker daemon http_proxy env var to ${DOCKER_HTTP_PROXY}"
        echo "export http_proxy=${DOCKER_HTTP_PROXY}" | \
            tee --append /etc/default/docker
    fi

    if [ ! -z "${DOCKER_HTTPS_PROXY}" ]; then
        echo "Setting docker daemon https_proxy env var to ${DOCKER_HTTPS_PROXY}"
        echo "export https_proxy=${DOCKER_HTTPS_PROXY}" | \
            tee --append /etc/default/docker
    fi
}

setup_ca(){
    if [ -f "${CA_CERT_FILE}" ]; then
        echo "Adding ${CA_CERT_FILE} as a trusted root CA"
        cp "${CA_CERT_FILE}" /usr/local/share/ca-certificates/

        update-ca-certificates
    fi
}

main(){
    setup_docker_proxy

    setup_ca

    /usr/local/bin/runner_orig.sh "${@}"
}

main "${@}"

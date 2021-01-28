# Copyright 2017 The Kubernetes Authors.
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

# Includes basic workspace setup, with gcloud and a bootstrap runner
FROM fedora:32

WORKDIR /workspace
RUN mkdir -p /workspace
ENV WORKSPACE=/workspace \
    TERM=xterm

# add env we can debug with the image name:tag
ARG IMAGE_ARG
ENV IMAGE=${IMAGE_ARG}

# Install packages
RUN dnf install -y dnf-plugins-core && \
    dnf -y install \
        java-11-openjdk-devel \
        cpio \
        patch \
        make \
        git \
        mercurial \
        sudo \
        gcc \
        gcc-c++ \
        glibc-static \
        libstdc++-static \
        glibc-devel \
        findutils \
        rsync-daemon \
        rsync \
        protobuf-compiler \
        python3-devel \
        redhat-rpm-config \
        jq \
        wget && \
    dnf -y clean all

# Install gcloud
ENV PATH=/google-cloud-sdk/bin:/workspace:${PATH} \
    CLOUDSDK_CORE_DISABLE_PROMPTS=1

RUN wget -q https://dl.google.com/dl/cloudsdk/channels/rapid/google-cloud-sdk.tar.gz && \
    tar xzf google-cloud-sdk.tar.gz -C / && \
    rm google-cloud-sdk.tar.gz && \
    /google-cloud-sdk/install.sh \
        --disable-installation-options \
        --bash-completion=false \
        --path-update=false \
        --usage-reporting=false && \
    gcloud components install alpha beta kubectl && \
    gcloud info | tee /workspace/gcloud-info.txt


#
# BEGIN: DOCKER IN DOCKER SETUP
#
# Install packages

RUN dnf install -y \
        moby-engine && \
    dnf -y clean all

# Move Docker's storage location
RUN echo 'DOCKER_OPTS="${DOCKER_OPTS} --data-root=/docker-graph"' | \
    tee --append /etc/default/docker
# NOTE this should be mounted and persisted as a volume ideally (!)
# We will make a fallback one now just in case
RUN mkdir /docker-graph

#
# END: DOCKER IN DOCKER SETUP
#

# note the runner is also responsible for making docker in docker function if
# env DOCKER_IN_DOCKER_ENABLED is set and similarly responsible for generating
# .bazelrc files if bazel remote caching is enabled 
COPY ["entrypoint.sh", "runner.sh", "create_bazel_cache_rcs.sh", \
        "/usr/local/bin/"]

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

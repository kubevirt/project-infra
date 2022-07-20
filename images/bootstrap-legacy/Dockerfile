# Copyright 2017 The Kubernetes Authors.
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

# Includes basic workspace setup, with gcloud and a bootstrap runner
FROM fedora:36

WORKDIR /workspace
RUN mkdir -p /workspace
ENV WORKSPACE=/workspace \
    TERM=xterm

# add env we can debug with the image name:tag
ARG IMAGE_ARG
ENV IMAGE=${IMAGE_ARG}

# Install packages
RUN dnf install -y \
    cpio \
    dnf-plugins-core \
    findutils \
    gcc \
    gcc-c++ \
    gettext \
    git \
    glibc-devel \
    glibc-static \
    iproute \
    java-11-openjdk-devel \
    jq \
    libstdc++-static \
    libvirt-devel \
    make \
    mercurial \
    patch \
    protobuf-compiler \
    python3-devel \
    python-unversioned-command \
    redhat-rpm-config \
    rsync \
    rsync-daemon \
    skopeo \
    sudo \
    buildah \
    qemu-user-static \
    wget &&\
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
        kmod \
        procps-ng \
        moby-engine && \
    dnf -y clean all

# Create directory for docker storage location
# NOTE this should be mounted and persisted as a volume ideally (!)
# We will make a fallback one now just in case
RUN mkdir /docker-graph

#
# END: DOCKER IN DOCKER SETUP
#

# Cache the most commonly used bazel versions in the container
RUN  curl -Lo ./bazelisk https://github.com/bazelbuild/bazelisk/releases/download/v1.7.4/bazelisk-linux-amd64 && \
     chmod +x ./bazelisk && mv ./bazelisk /usr/local/bin/bazelisk && \
     cd /usr/local/bin && ln -s bazelisk bazel

# Cache the most commonly used bazel versions inside the image
# and remove resulting cache directories to save a few hundred MB
RUN USE_BAZEL_VERSION=4.1.0 bazel version && \
    USE_BAZEL_VERSION=4.2.1 bazel version && \
    rm -rf /root/.cache/bazel

# create mixin directories
RUN mkdir -p /etc/setup.mixin.d/ && mkdir -p /etc/teardown.mixin.d/

# Trust git repositories used for e2e jobs
RUN git config --global --add safe.directory '*'

# note the runner is also responsible for making docker in docker function if
# env DOCKER_IN_DOCKER_ENABLED is set and similarly responsible for generating
# .bazelrc files if bazel remote caching is enabled
COPY ["entrypoint.sh", "runner.sh", "create_bazel_cache_rcs.sh", \
        "/usr/local/bin/"]

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

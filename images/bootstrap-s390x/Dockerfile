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
FROM fedora:39

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
    procps-ng \
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
    bind-utils \
    wget \
    python3-jinja2 &&\
  dnf -y clean all

# BEGIN: PODMAN IN CONTAINER SETUP
#

RUN dnf -y update; dnf -y reinstall shadow-utils && \
    dnf -y install crun podman --exclude container-selinux && \
    dnf -y clean all && \
    ln -s /usr/bin/podman /usr/bin/docker

VOLUME /var/lib/containers

COPY storage.conf /etc/containers/storage.conf
RUN mkdir -p /var/lib/shared-images

COPY containers.conf /etc/containers/containers.conf
COPY podman.json /etc/containers/networks/podman.json

RUN chmod 644 /etc/containers/containers.conf

ENV _CONTAINERS_USERNS_CONFIGURED=""

#
# END: PODMAN IN CONTAINER SETUP
#


# create mixin directories
RUN mkdir -p /etc/setup.mixin.d/ && mkdir -p /etc/teardown.mixin.d/

# Trust git repositories used for e2e jobs
RUN git config --global --add safe.directory '*'

# note the runner is also responsible for making podman in container function if
# env PODMAN_IN_CONTAINER_ENABLED is set and similarly responsible for generating

# .bazelrc files if bazel remote caching is enabled
COPY ["entrypoint.sh", "runner.sh", "create_bazel_cache_rcs.sh", \
        "/usr/local/bin/"]

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]

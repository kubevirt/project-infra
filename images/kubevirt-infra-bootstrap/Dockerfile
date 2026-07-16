FROM quay.io/kubevirtci/bootstrap:v20260715-af3e234

RUN dnf update -y \
    && dnf install -y \
        ansible \
        expect \
        git \
        intltool \
        libosinfo \
        python3-devel \
        openssl-devel\
        make \
        osinfo-db-tools \
        python3-gobject \
        python3 \
        python3-pip \
        python3-yaml \
        rsync \
        podman \
        && dnf clean all -y

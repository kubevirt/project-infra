FROM quay.io/kubevirtci/bootstrap:v20220405-a56be88

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
RUN export KUSTOMIZE_DIR=/opt/kustomize \
    && mkdir -p $KUSTOMIZE_DIR \
    && cd $KUSTOMIZE_DIR \
    && wget "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv3.5.4/kustomize_v3.5.4_linux_amd64.tar.gz" \
    && tar xzf ./kustomize_v3.5.4_linux_amd64.tar.gz \
    && rm kustomize_v3.5.4_linux_amd64.tar.gz \
    && ln -s $KUSTOMIZE_DIR/kustomize /usr/bin/kustomize

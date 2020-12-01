FROM kubevirtci/bootstrap:v20201119-a5880e0

RUN apt-get update \
    && apt-get install -y --no-install-recommends --no-upgrade \
        ansible \
        expect \
        git \
        intltool \
        libosinfo-1.0 \
        libpython-dev \
        libssl-dev \
        make \
        osinfo-db-tools \
        python-gi \
        python3 \
        python3-pip \
        python3-yaml \
        rsync \
        && rm -rf /var/lib/apt/lists/*
RUN export KUSTOMIZE_DIR=/opt/kustomize \
    && mkdir -p $KUSTOMIZE_DIR \
    && cd $KUSTOMIZE_DIR \
    && wget "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv3.5.4/kustomize_v3.5.4_linux_amd64.tar.gz" \
    && tar xzf ./kustomize_v3.5.4_linux_amd64.tar.gz \
    && rm kustomize_v3.5.4_linux_amd64.tar.gz \
    && ln -s $KUSTOMIZE_DIR/kustomize /usr/bin/kustomize

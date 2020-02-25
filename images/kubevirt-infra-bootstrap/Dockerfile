FROM gcr.io/k8s-testimages/bootstrap:v20190516-c6832d9

RUN apt-get update \
    && apt-get install -y --no-install-recommends --no-upgrade \
        ansible \
        intltool \
        osinfo-db-tools \
        libosinfo-1.0 \
        expect python-gi \
        python3-yaml \
        make \
        rsync \
        git \
        libssl-dev \
        libpython-dev \
        python3 \
        python3-pip \
        && rm -rf /var/lib/apt/lists/*
RUN export KUSTOMIZE_DIR=/opt/kustomize \
    && mkdir -p $KUSTOMIZE_DIR \
    && cd $KUSTOMIZE_DIR \
    && wget "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv3.5.4/kustomize_v3.5.4_linux_amd64.tar.gz" \
    && tar xzf ./kustomize_v3.5.4_linux_amd64.tar.gz \
    && rm kustomize_v3.5.4_linux_amd64.tar.gz \
    && echo "export PATH=$KUSTOMIZE_DIR:$PATH" >> $HOME/.bashrc

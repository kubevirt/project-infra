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
        && rm -rf /var/lib/apt/lists/*

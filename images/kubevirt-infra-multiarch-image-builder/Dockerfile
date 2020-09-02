FROM gcr.io/k8s-testimages/bootstrap:v20190516-c6832d9

RUN apt update && apt install -y wget git python python3 \
    && wget https://github.com/bazelbuild/bazelisk/releases/download/v1.6.1/bazelisk-linux-amd64 -O /usr/local/bin/bazelisk \
    && chmod +x /usr/local/bin/bazelisk \
    && rm -rf /var/lib/apt/lists/*

RUN git clone https://github.com/kubernetes/test-infra.git /test-infra && cd /test-infra && git checkout 686fce5576ff231655b7374f9a2b3f9c864c54e1

ENV USE_BAZEL_VERSION=3.0.0

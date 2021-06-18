FROM quay.io/kubevirtci/bootstrap:v20210126-a12b6c0

RUN git clone https://github.com/kubernetes/test-infra.git && \
  cd test-infra && \
  git checkout e88598c4a7f86e0564a2d8b46ce5f729247791e6 && \
  bazelisk build //prow/cmd/config-bootstrapper && \
  cp bazel-bin/prow/cmd/config-bootstrapper/linux_amd64_stripped/config-bootstrapper /usr/local/bin && \
  bazelisk build //prow/cmd/phony && \
  cp bazel-bin/prow/cmd/phony/linux_amd64_stripped/phony /usr/local/bin && \
  bazelisk clean --expunge && \
  cd .. && rm -rf test-infra && \
  rm -rf ~/.cache.bazel

RUN apt-get update && apt-get install -y --no-install-recommends \
  python3-venv

RUN curl -Lo ./kustomize.tar.gz https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v3.8.7/kustomize_v3.8.7_linux_amd64.tar.gz && \
  tar -xvf kustomize.tar.gz && \
  mv ./kustomize /usr/local/bin && \
  rm kustomize.tar.gz

RUN curl -Lo ./yq https://github.com/mikefarah/yq/releases/download/3.4.1/yq_linux_amd64 && \
  chmod a+x ./yq && \
  mv ./yq /usr/local/bin

RUN curl -Lo ./kind https://github.com/kubernetes-sigs/kind/releases/download/v0.11.1/kind-linux-amd64 && \
  chmod a+x ./kind && \
  mv ./kind /usr/local/bin

RUN curl -LO "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl" && \
  chmod a+x ./kubectl && \
  mv ./kubectl /usr/local/bin

COPY requirements.txt .

RUN pip install -r requirements.txt

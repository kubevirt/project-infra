FROM quay.io/kubevirtci/bootstrap:v20210906-994b913

RUN git clone https://github.com/kubernetes/test-infra.git && \
  cd test-infra && \
  git checkout 689941423e01abb85b7ca6b9e317e3d035c60098 && \
  bazelisk build //prow/cmd/config-bootstrapper && \
  cp bazel-bin/prow/cmd/config-bootstrapper/config-bootstrapper_/config-bootstrapper /usr/local/bin && \
  config-bootstrapper --help && \
  bazelisk build //prow/cmd/phony && \
  cp bazel-bin/prow/cmd/phony/phony_/phony /usr/local/bin && \
  phony --help && \
  bazelisk build //prow/cmd/hmac && \
  cp bazel-bin/prow/cmd/hmac/hmac_/hmac /usr/local/bin && \
  hmac --help && \
  bazelisk clean --expunge && \
  cd .. && rm -rf test-infra && \
  rm -rf ~/.cache.bazel

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

RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
  chmod a+x ./kubectl && \
  mv ./kubectl /google-cloud-sdk/bin/ && \
  kubectl version --client=true

COPY requirements.txt .

RUN pip install -r requirements.txt

RUN dnf install -y which

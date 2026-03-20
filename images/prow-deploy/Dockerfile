FROM quay.io/kubevirtci/golang:v20260319-c8f1db8 as builder

RUN git clone --depth 1 --revision 3815e13b727dd09738bf9bd7e8a06ab616cb0d62 https://github.com/kubernetes/test-infra.git && \
  cd test-infra && \
  /usr/local/bin/runner.sh /bin/sh -ce 'go mod vendor' && \
  /usr/local/bin/runner.sh /bin/sh -ce 'go build -o /usr/local/bin/config-bootstrapper ./prow/cmd/config-bootstrapper/...' && \
  /usr/local/bin/config-bootstrapper --help && \
  /usr/local/bin/runner.sh /bin/sh -ce 'go build -o /usr/local/bin/phony ./prow/cmd/phony' && \
  /usr/local/bin/phony --help && \
  /usr/local/bin/runner.sh /bin/sh -ce 'go build -o /usr/local/bin/hmac ./prow/cmd/hmac' && \
  /usr/local/bin/hmac --help

FROM quay.io/kubevirtci/bootstrap:v20260319-c8f1db8

COPY --from=builder /usr/local/bin/config-bootstrapper /usr/local/bin/phony /usr/local/bin/hmac /usr/local/bin/

RUN curl -Lo ./kustomize.tar.gz https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v3.8.7/kustomize_v3.8.7_linux_amd64.tar.gz && \
  tar -xvf kustomize.tar.gz && \
  mv ./kustomize /usr/local/bin && \
  rm kustomize.tar.gz

# TODO: remove yq installation after golang image tag bump
RUN curl -fsSLo ./yq https://github.com/mikefarah/yq/releases/download/v4.47.1/yq_linux_amd64 && \
  chmod +x ./yq && mv ./yq /usr/local/bin

RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
  chmod a+x ./kubectl && \
  mv ./kubectl /google-cloud-sdk/bin/ && \
  /google-cloud-sdk/bin/kubectl version --client=true

COPY requirements.txt .

RUN pip install -r requirements.txt

RUN dnf install -y which openssl

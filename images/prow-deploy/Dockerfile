FROM quay.io/kubevirtci/golang:v20260417-7427ea2 as builder

RUN git clone --depth 1 --revision 3815e13b727dd09738bf9bd7e8a06ab616cb0d62 https://github.com/kubernetes/test-infra.git && \
  cd test-infra && \
  /usr/local/bin/runner.sh /bin/sh -ce 'time go build -o /usr/local/bin/phony ./prow/cmd/phony' && \
  /usr/local/bin/phony --help

FROM gcr.io/k8s-prow/config-bootstrapper:v20221117-3815e13b72 as cfg-bootstrap
FROM gcr.io/k8s-prow/hmac:v20221117-3815e13b72 as hmac

FROM quay.io/kubevirtci/bootstrap:v20260417-7427ea2

COPY --from=builder /usr/local/bin/phony /usr/local/bin/
COPY --from=cfg-bootstrap /ko-app/config-bootstrapper /usr/local/bin/
COPY --from=hmac /ko-app/hmac /usr/local/bin/

RUN curl -fLo ./kustomize.tar.gz https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v3.8.7/kustomize_v3.8.7_linux_amd64.tar.gz && \
  tar -xvf kustomize.tar.gz && \
  mv ./kustomize /usr/local/bin && \
  rm kustomize.tar.gz

RUN curl -fLRO "https://dl.k8s.io/release/$(curl -fsSL https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
  chmod a+x ./kubectl && \
  mv ./kubectl /google-cloud-sdk/bin/ && \
  /google-cloud-sdk/bin/kubectl version --client=true

RUN dnf install -y ansible-core && \
    dnf clean all

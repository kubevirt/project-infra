FROM quay.io/kubevirtci/golang:v20240814-9b6c4af

ENV GIMME_GO_VERSION=1.22.6

RUN set -x && \
    git clone https://github.com/kubernetes/test-infra.git && \
    cd ./test-infra && \
    git checkout e4d87dae72dbf1754625cbf112afc535307c1db4 && \
    /usr/local/bin/runner.sh /bin/sh -ce 'go build -o ./robots/pr-creator/pr-creator ./robots/pr-creator/main.go' && \
    mv ./robots/pr-creator/pr-creator /usr/local/bin/ && \
    chmod +x /usr/local/bin/pr-creator && \
    /usr/local/bin/runner.sh /bin/sh -ce 'go clean -cache -modcache' && \
    cd .. && rm -rf ./test-infra

RUN set -x && \
    git clone https://github.com/kubevirt/project-infra.git && \
    cd ./project-infra && \
    cp ./hack/git-askpass.sh /usr/local/bin && \
    cp ./hack/git-pr.sh /usr/local/bin && \
    chmod +x /usr/local/bin/git-askpass.sh /usr/local/bin/git-pr.sh && \
    /usr/local/bin/runner.sh /bin/sh -ce 'go build -o /usr/local/bin/labels-checker ./robots/cmd/labels-checker' && \
    chmod +x /usr/local/bin/labels-checker && \
    cd .. && rm -rf ./project-infra

RUN dnf install -y \
        skopeo && \
    dnf -y clean all

ENV GIT_ASKPASS=/usr/local/bin/git-askpass.sh

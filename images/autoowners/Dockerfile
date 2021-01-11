FROM golang:1.13 as builder

WORKDIR /go/src/github.com/openshift/ci-tools/
RUN mkdir -p /go/src/github.com/openshift/ && \
    cd /go/src/github.com/openshift/ && \
    git clone https://github.com/openshift/ci-tools.git && \
    cd ci-tools/ && \
    export GOPROXY=off && \
    export GOFLAGS=-mod=vendor && \
    go install ./cmd/autoowners/...

FROM centos:7

COPY --from=builder /go/bin/autoowners /usr/bin/autoowners

RUN yum install -y git

# change ownership to allow access
# when selinux is enabled
RUN mkdir -p /etc/github && \
    chgrp -R 0 /etc/github && \
    chmod -R g+rwX /etc/github

ENTRYPOINT ["/usr/bin/autoowners"]

FROM golang:1.13 as builder

WORKDIR /go/src/github.com/openshift/ci-tools/
RUN mkdir -p /go/src/github.com/openshift/ && \
    cd /go/src/github.com/openshift/ && \
    git clone https://github.com/openshift/ci-tools.git && \
    cd ci-tools/ && \
    export GOPROXY=off && \
    export GOFLAGS=-mod=vendor && \
    env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/autoowners ./cmd/autoowners/...

FROM gcr.io/k8s-prow/git:v20200605-44f6c96

COPY --from=builder /go/bin/autoowners /usr/bin/autoowners

ENTRYPOINT ["/usr/bin/autoowners"]

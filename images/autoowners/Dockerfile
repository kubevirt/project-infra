FROM golang:1.18 as builder

WORKDIR /go/src/github.com/openshift/ci-tools/
RUN mkdir -p /go/src/github.com/openshift/ && \
    cd /go/src/github.com/openshift/ && \
    git clone https://github.com/openshift/ci-tools.git && \
    cd ci-tools/ && \
    env GOPROXY=off GOFLAGS=-mod=vendor CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/autoowners ./cmd/autoowners/...

FROM gcr.io/k8s-prow/git:v20220523-6026203ca9

COPY --from=builder /go/bin/autoowners /usr/bin/autoowners

ENTRYPOINT ["/usr/bin/autoowners"]

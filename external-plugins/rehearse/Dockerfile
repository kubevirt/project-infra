FROM golang:1.14.6-alpine3.12 as builder

RUN mkdir /build
COPY plugin/ /build
RUN cd /build/ && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -tags netgo -ldflags '-w -extldflags "-static"' -v .

FROM alpine/git:v2.26.2
COPY --from=builder /build/rehearse /bin/rehearse
ENTRYPOINT ["/bin/rehearse"]

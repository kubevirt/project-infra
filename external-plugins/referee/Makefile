
.PHONY: all format push
all: format push
bazelbin := bazelisk

build:
	$(bazelbin) build //external-plugins/referee/...

format:
	gofmt -w .

push:
	podman build -f ../../images/referee/Containerfile -t quay.io/kubevirtci/referee:latest && podman push quay.io/kubevirtci/referee:latest


.PHONY: all build test format push
all: format push
bazelbin := bazelisk
CONTAINER_TAG := $(shell ../../hack/container-tag.sh)

build:
	$(bazelbin) build //external-plugins/referee/...

test:
	$(bazelbin) test //external-plugins/referee/...

format:
	gofmt -w .

push:
	podman build -f ../../images/referee/Containerfile -t quay.io/kubevirtci/referee:$(CONTAINER_TAG) && podman push quay.io/kubevirtci/referee:$(CONTAINER_TAG)

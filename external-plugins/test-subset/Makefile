
.PHONY: all build test format push
all: format push
CONTAINER_TAG := $(shell ../../hack/container-tag.sh)
CONTAINER_IMAGE := test-subset
CONTAINER_REPO := quay.io/kubevirtci/$(CONTAINER_IMAGE)

build:
	go build ./...

test:
	go test -v ./...

format:
	gofmt -w .

push:
	podman build -f ../../images/$(CONTAINER_IMAGE)/Containerfile -t $(CONTAINER_REPO):$(CONTAINER_TAG) && podman push $(CONTAINER_REPO):$(CONTAINER_TAG)
	bash -x ../../hack/update-deployments-with-latest-image.sh $(CONTAINER_REPO) $(CONTAINER_TAG)

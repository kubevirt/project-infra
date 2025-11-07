
.PHONY: all build test format push
all: format push
CONTAINER_TAG := $(shell ../../hack/container-tag.sh)
CONTAINER_IMAGE := referee
CONTAINER_REPO := quay.io/kubevirtci/$(CONTAINER_IMAGE)

build:
	go build ./...

test:
	go test ./...

format:
	gofmt -w .

push:
	cd ../../images && ./publish_image.sh -e $(CONTAINER_IMAGE) quay.io kubevirtci
	bash -x ../../hack/update-deployments-with-latest-image.sh $(CONTAINER_REPO)


.PHONY: all build test format push
all: format push
bazelbin := bazelisk
CONTAINER_TAG := $(shell ../../hack/container-tag.sh)
CONTAINER_IMAGE := referee
CONTAINER_REPO := quay.io/kubevirtci/$(CONTAINER_IMAGE)

build:
	$(bazelbin) build //external-plugins/referee/...

test:
	$(bazelbin) test //external-plugins/referee/...

format:
	gofmt -w .

push:
	podman build -f ../../images/$(CONTAINER_IMAGE)/Containerfile -t $(CONTAINER_REPO):$(CONTAINER_TAG) && podman push $(CONTAINER_REPO):$(CONTAINER_TAG)
	bash -x ../../hack/update-deployments-with-latest-image.sh $(CONTAINER_REPO)

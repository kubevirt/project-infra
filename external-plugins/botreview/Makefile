
.PHONY: all clean format test push
all: format test push
bazelbin := bazelisk
CONTAINER_TAG := $(shell ../../hack/container-tag.sh)
CONTAINER_IMAGE := botreview
CONTAINER_REPO := quay.io/kubevirtci/$(CONTAINER_IMAGE)

build:
	$(bazelbin) build //external-plugins/botreview/...

format:
	gofmt -w .

test:
	$(bazelbin) test //external-plugins/botreview/...

push:
	podman build -f ../../images/$(CONTAINER_IMAGE)/Containerfile -t $(CONTAINER_REPO):$(CONTAINER_TAG) && podman push $(CONTAINER_REPO):$(CONTAINER_TAG)
	bash -x ../../hack/update-deployments-with-latest-image.sh $(CONTAINER_REPO)

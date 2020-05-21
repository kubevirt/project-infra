
.PHONY: all build build-container push-container test
all: build

IMAGE_BUILD_CMD ?= "docker"
IMAGE_REGISTRY ?= "quay.io"
REGISTRY_NAMESPACE ?= ""
IMAGE_TAG ?= "latest"
IMAGE_NAME ?= "release-blocker"

FULL_IMAGE ?= "$(IMAGE_REGISTRY)/$(REGISTRY_NAMESPACE)/$(IMAGE_NAME):$(IMAGE_TAG)"

test:
	go test

build:
	go build

build-container: build
	@echo "Building the release-blocker prow plugin image"
	@if [ -z "$(REGISTRY_NAMESPACE)" ]; then\
		echo "REGISTRY_NAMESPACE env-var must be set to your $(IMAGE_REGISTRY) namespace";\
		exit 1;\
	fi
	$(IMAGE_BUILD_CMD) build --no-cache -f Dockerfile -t $(FULL_IMAGE) .

push-container:
	$(IMAGE_BUILD_CMD) push $(FULL_IMAGE)


CONTAINER_IMAGE := phased
CONTAINER_REPO := quay.io/kubevirtci/$(CONTAINER_IMAGE)

.PHONY: all clean format test push
all: format test push

build:
	go build ./plugin/...

format:
	gofmt -w .

test:
	go test ./plugin/...

push:
	cd ../../images && ./publish_image.sh $(CONTAINER_IMAGE) quay.io kubevirtci
	bash -x ../../hack/update-deployments-with-latest-image.sh $(CONTAINER_REPO)

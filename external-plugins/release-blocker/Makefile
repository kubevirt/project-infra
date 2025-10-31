CONTAINER_IMAGE := release-blocker
CONTAINER_REPO := quay.io/kubevirtci/$(CONTAINER_IMAGE)

.PHONY: all clean format test push
all: format test push

build:
	go build ./...

format:
	gofmt -w .

test:
	go test ./...

push:
	cd ../../images && ./publish_image.sh -e $(CONTAINER_IMAGE) quay.io kubevirtci
	bash -x ../../hack/update-jobs-with-latest-image.sh $(CONTAINER_REPO)

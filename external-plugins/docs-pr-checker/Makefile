CONTAINER_IMAGE := docs-pr-checker
CONTAINER_REPO := quay.io/kubevirtci/$(CONTAINER_IMAGE)

.PHONY: all clean format test build push
all: format test build

build:
	go build ./...

format:
	gofmt -w .

test:
	go test ./...

push:
	cd ../../images && ./publish_image.sh $(CONTAINER_IMAGE) quay.io kubevirtci
	bash -x ../../hack/update-jobs-with-latest-image.sh $(CONTAINER_REPO)

clean:
	go clean


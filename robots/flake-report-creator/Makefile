CONTAINER_IMAGE := flake-report-creator
CONTAINER_REPO := quay.io/kubevirtci/$(CONTAINER_IMAGE)

.PHONY: all clean verify format test
all: format verify test push

format:
	gofmt -w .

test:
	go test ./... ../pkg/flakefinder/...:*

verify:
	../build-verify.sh

push:
	cd ../../images && ./publish_image.sh $(CONTAINER_IMAGE) quay.io kubevirtci
	bash -x "../../hack/update-jobs-with-latest-image.sh" "$(CONTAINER_REPO)"
	bash -x "../../hack/update-scripts-with-latest-image.sh" "$(CONTAINER_REPO)"

clean:
	go clean -cache -modcache

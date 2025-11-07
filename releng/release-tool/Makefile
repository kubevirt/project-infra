.PHONY: all format build container-base-build container-base-push container-build container-push test
all: format build

format:
	gofmt -w .

build: 
	go build .

test:
	go test

container-build: test
	( cd ../../images && ./publish_image.sh -b -e release-tool quay.io kubevirtci )

container-push: container-push-image container-update-latest-tag

container-push-image:
	( cd ../../images && ./publish_image.sh -e release-tool quay.io kubevirtci )

container-update-latest-tag:
	( cd ../../images && ./update-latest-image-tag.sh release-tool quay.io kubevirtci )

container-base-build:
	( cd ../../images && ./publish_image.sh -b release-tool-base quay.io kubevirtci )

container-base-push:
	( cd ../../images && ./publish_image.sh release-tool-base quay.io kubevirtci )

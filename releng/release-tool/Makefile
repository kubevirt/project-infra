.PHONY: all format build container-base-build container-base-push container-build container-push test
all: format build

format:
	gofmt -w .

build: 
	go build .

test:
	go test

container-build: test
	( cd ../../images && ./publish_image.sh -b release-tool quay.io kubevirtci )

container-push:
	( cd ../../images && ./publish_image.sh release-tool quay.io kubevirtci )

container-base-build:
	( cd ../../images && ./publish_image.sh -b release-tool-base quay.io kubevirtci )

container-base-push:
	( cd ../../images && ./publish_image.sh release-tool-base quay.io kubevirtci )

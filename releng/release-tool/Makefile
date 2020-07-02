.PHONY: all format build container-base-build container-base-push container-build container-push test
all: format build
bazelbin := bazelisk
date_prefix := `date +"v20%y%m%d-"`
hash := `git rev-parse HEAD`

format:
	gofmt -w .

build: 
	go build .

test:
	go test

container-build: test
	$(bazelbin) run //releng/release-tool:release-tool-image

container-push:
	$(bazelbin) run //releng/release-tool:release-tool-push

container-base-build:
	docker build . -f Dockerfile.release-tool-base -t kubevirtci/release-tool-base:$(date_prefix)$(hash)

container-base-push:
	docker push kubevirtci/release-tool-base:$(date_prefix)$(hash)

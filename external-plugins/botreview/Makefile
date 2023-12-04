
.PHONY: all clean format test push
all: format test push
bazelbin := bazelisk

build:
	$(bazelbin) build //external-plugins/botreview/...

format:
	gofmt -w .

test:
	$(bazelbin) test //external-plugins/botreview/...

push:
	podman build -f ./images/botreview/Containerfile -t quay.io/kubevirtci/botreview:latest && podman push quay.io/kubevirtci/botreview:latest

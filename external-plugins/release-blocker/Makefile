
.PHONY: all clean format test push
all: format test push
bazelbin := bazelisk

build:
	$(bazelbin) build //external-plugins/release-blocker:release-blocker

format:
	gofmt -w .

test:
	$(bazelbin) test //external-plugins/release-blocker:*

push:
	$(bazelbin) run //external-plugins/release-blocker:push
	bash -x ../../hack/update-jobs-with-latest-image.sh quay.io/kubevirtci/release-blocker

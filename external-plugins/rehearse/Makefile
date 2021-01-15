
.PHONY: all clean format test push
all: format test push
bazelbin := bazelisk

build:
	$(bazelbin) build //external-plugins/release/plugin

format:
	gofmt -w .

test:
	$(bazelbin) test //external-plugins/release/plugin:*

push:
	$(bazelbin) run //external-plugins/release/plugin:push
	bash -x ../../hack/update-jobs-with-latest-image.sh kubevirtci/release

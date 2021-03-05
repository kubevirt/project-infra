.PHONY: all clean format test push
all: format build push
bazelbin := bazelisk

format:
	gofmt -w .

build:
	$(bazelbin) build //robots/cmd/indexpagecreator:*

push:
	$(bazelbin) run //robots/cmd/indexpagecreator:push
	bash -x ../../../hack/update-jobs-with-latest-image.sh kubevirtci/indexpagecreator

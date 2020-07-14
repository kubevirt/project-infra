.PHONY: all clean format test push
all: format build push
bazelbin := bazelisk

format:
	gofmt -w .

build:
	$(bazelbin) build //robots/indexpagecreator:*

push:
	$(bazelbin) run //robots/indexpagecreator:push
	bash -x ./update-jobs-with-latest-indexpagecreator-image.sh


.PHONY: all clean format test push
all: format test push
bazelbin := bazelisk

build:
	$(bazelbin) build //external-plugins/rehearse/plugin

format:
	gofmt -w .

test:
	$(bazelbin) test //external-plugins/rehearse/plugin:*

push:
	$(bazelbin) run //external-plugins/rehearse/plugin:push
	bash -x ../../hack/update-jobs-with-latest-image.sh kubevirtci/rehearse

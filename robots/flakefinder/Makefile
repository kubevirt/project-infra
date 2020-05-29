.PHONY: all clean format test push
all: format test push
bazelbin := bazelisk

format:
	gofmt -w .

test:
	$(bazelbin) test //robots/flakefinder:*

push:
	$(bazelbin) run //robots/flakefinder:push
	bash -x ./update-jobs-with-latest-flakefinder-image.sh

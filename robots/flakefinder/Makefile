.PHONY: all clean format test push
all: format test push

format:
	gofmt -w .

test:
	bazel test //robots/flakefinder:*

push:
	push.sh

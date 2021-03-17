.PHONY: all clean format test push
all: format test push

format:
	gofmt -w .

test:
	bazel test //robots/cmd/flakefinder:*

push:
	../push.sh flakefinder

clean:
	bazel clean --expunge

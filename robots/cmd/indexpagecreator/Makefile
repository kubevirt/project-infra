.PHONY: all clean verify format test push
all: format verify test push

format:
	gofmt -w .

test:
	bazel build //robots/cmd/indexpagecreator:* //robots/pkg/flakefinder:*

verify:
	../build-verify.sh

push:
	../push.sh indexpagecreator

clean:
	bazel clean --expunge

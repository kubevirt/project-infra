.PHONY: all clean verify format test push
all: format verify test push

format:
	gofmt -w .

test:
	bazel test //robots/cmd/flakefinder:* //robots/pkg/flakefinder/...:*

verify:
	../build-verify.sh

push:
	../push.sh flakefinder

clean:
	bazel clean --expunge

.PHONY: all clean verify format test push
all: format verify test push

format:
	gofmt -w .

test:
	bazel test //robots/cmd/flake-issue-creator:*

verify:
	../build-verify.sh

push:
	../push.sh flake-issue-creator

clean:
	bazel clean --expunge

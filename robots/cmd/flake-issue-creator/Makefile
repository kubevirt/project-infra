.PHONY: all clean format test push
all: format test push

format:
	gofmt -w .

test:
	bazel test //robots/cmd/flake-issue-creator:*

push:
	../push.sh flake-issue-creator

clean:
	bazel clean --expunge

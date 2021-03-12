.PHONY: all clean format test push
all: format test push

format:
	gofmt -w .

test:
	bazel test --define container_tag= //robots/cmd/flake-report-writer:*

push:
	push.sh

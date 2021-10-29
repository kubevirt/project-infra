.PHONY: all clean verify format test
all: format verify test

format:
	gofmt -w .

test:
	bazel test //robots/cmd/flake-usage-creator:* //robots/pkg/flakefinder:*

verify:
	../build-verify.sh

clean:
	bazel clean --expunge

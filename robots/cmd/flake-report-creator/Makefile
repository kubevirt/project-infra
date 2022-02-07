.PHONY: all clean verify format test
all: format verify test push

format:
	gofmt -w .

test:
	bazel test //robots/cmd/flake-report-creator/...:* //robots/pkg/flakefinder/...:*

verify:
	../build-verify.sh

push:
	../push.sh flake-report-creator

clean:
	bazel clean --expunge

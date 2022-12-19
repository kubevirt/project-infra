.PHONY: all clean verify format test
all: format verify test push

format:
	gofmt -w .

test:
	bazel test //robots/cmd/test-report/...:* //robots/pkg/test-report/...:* //robots/pkg/flakefinder/...:* //robots/pkg/jenkins/...:*

verify:
	../build-verify.sh

push:
	../push.sh test-report

clean:
	bazel clean --expunge

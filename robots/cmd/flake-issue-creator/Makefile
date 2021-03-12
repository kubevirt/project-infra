.PHONY: all clean format test push
all: format test push

format:
	gofmt -w .

test:
	bazel test --define container_tag= //robots/cmd/flake-issue-creator:*

push:
	bazel run --define container_tag=$(./get-image-tag.sh) //robots/cmd/flake-issue-creator:push
	bash -x ../../../hack/update-jobs-with-latest-image.sh quay.io/kubevirtci/flake-issue-creator

clean:
	bazel clean --expunge

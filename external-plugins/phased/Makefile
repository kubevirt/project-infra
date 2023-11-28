
.PHONY: all clean format test push
all: format test push
bazelbin := bazelisk

build:
	$(bazelbin) build //external-plugins/phased/plugin

format:
	gofmt -w .

test:
	$(bazelbin) test //external-plugins/phased/...

push:
	$(bazelbin) run --stamp --workspace_status_command="./hack/print-workspace-status-no-git-tag.sh" //external-plugins/phased/plugin:push
	bash -x ../../hack/update-deployments-with-latest-image.sh quay.io/kubevirtci/phased

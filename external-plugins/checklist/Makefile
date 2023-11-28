
.PHONY: all clean format test push
all: format test push
bazelbin := bazelisk

build:
	$(bazelbin) build //external-plugins/checklist/plugin

format:
	gofmt -w .

test:
	$(bazelbin) test //external-plugins/checklist/...

push:
	$(bazelbin) run --stamp --workspace_status_command="./hack/print-workspace-status-no-git-tag.sh" //external-plugins/checklist/plugin:push
	bash -x ../../hack/update-deployments-with-latest-image.sh quay.io/kubevirtci/checklist

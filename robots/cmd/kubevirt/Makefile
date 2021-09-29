.PHONY: all format validate
all: gazelle test validate

build:
	bazel build //robots/cmd/kubevirt

test:
	bazel test --test_output=all -- //robots/pkg/kubevirt/...

format:
	gofmt -w . ../../pkg/kubevirt
	goimports -w -local kubevirt.io/project-infra . ../../pkg/kubevirt

gazelle:
	gazelle update
	cd ../../pkg/kubevirt && gazelle update

validate: format
	../../../hack/check-workspace-dirty.sh

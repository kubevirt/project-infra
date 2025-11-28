.PHONY: all format validate
all: gazelle test validate

build:
	go build ./...

test:
	go test ./... ../../pkg/kubevirt/...

format:
	gofmt -w . ../../pkg/kubevirt
	goimports -w -local kubevirt.io/project-infra . ../../pkg/kubevirt

validate: format
	../../../hack/check-workspace-dirty.sh

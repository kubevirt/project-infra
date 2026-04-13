.PHONY: all format validate
all: format test validate

test:
	go test ./... ../../pkg/imagebump/...

format:
	gofmt -w . ../../pkg/imagebump
	goimports -w -local kubevirt.io/project-infra . ../../pkg/imagebump

validate: format
	../../hack/check-workspace-dirty.sh

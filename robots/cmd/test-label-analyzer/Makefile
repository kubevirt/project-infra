.PHONY: all clean verify format test push
all: format test build

format:
	gofmt -w .

test:
	go test -v ./... ../../pkg/test-label-analyzer/...

build:
	go build -v ./...

.PHONY: all clean verify format test push
all: format test build

format:
	gofmt -w .

test:
	go test ./robots/test-label-analyzer/... ./pkg/test-label-analyzer/... ./pkg/git/...

build:
	go build ./robots/test-label-analyzer/...

.PHONY: all clean verify format test push
all: format test build

format:
	gofmt -w .

test:
	go test ./robots/cmd/test-label-analyzer/... ./robots/pkg/test-label-analyzer/... ./robots/pkg/git/...

build:
	go build ./robots/cmd/test-label-analyzer/...

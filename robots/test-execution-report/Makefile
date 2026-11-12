.PHONY: all clean verify format test

all: format verify test

format:
	gofmt -w .

test:
	go test ./...

verify:
	go build .
	go vet ./...

clean:
	go clean -cache -modcache

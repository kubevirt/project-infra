limiter     := limiter
flake-report-writer := robots/flake-report-writer
flake-issue-creator := robots/flake-issue-creator
querier := robots/release-querier
kubevirtci := robots/kubevirtci-bumper

ifndef ARTIFACTS
	ARTIFACTS=/tmp/artifacts
	export ARTIFACTS
endif
ifndef COVERAGE_OUTPUT_PATH
	COVERAGE_OUTPUT_PATH=${ARTIFACTS}/coverage.html
	export COVERAGE_OUTPUT_PATH
endif
ifndef WHAT_COVERAGE
	WHAT_COVERAGE=./external-plugins/... ./releng/... ./robots/...
	export WHAT_COVERAGE
endif

.PHONY: all clean deps-update update-labels install-metrics-binaries lint $(limiter) $(flake-report-writer) $(querier) $(kubevirtci) $(flake-issue-creator)
all: deps-update $(limiter) $(flake-report-writer) $(querier) $(kubevirtci) $(flake-issue-creator)

lint-clean:
	golangci-lint cache clean

clean: install-metrics-binaries lint-clean
	go clean -cache -modcache

$(limiter) $(flake-report-writer) $(querier) $(kubevirtci) $(flake-issue-creator): deps-update
	$(MAKE) --directory=$@

deps-update:
	export GO111MODULE=on
	go get ./...
	go mod tidy
	go mod vendor

build:
	go build ./external-plugins/... ./releng/... ./robots/... ./github/ci/services/...

test: build
	go test ./external-plugins/... ./releng/... ./robots/...

update-labels:
	./hack/labels/update.sh

lint:
	./hack/lint.sh

coverage:
	if ! command -V covreport; then go install github.com/cancue/covreport@latest; fi
	go test ${WHAT_COVERAGE} -coverprofile=/tmp/coverage.out
	mkdir -p ${ARTIFACTS}
	covreport  -i /tmp/coverage.out -o ${COVERAGE_OUTPUT_PATH}

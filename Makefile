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
ifndef COVERAGE_TARGETS
	COVERAGE_TARGETS=./external-plugins/... ./releng/... ./robots/... ./cmd/... ./pkg/...
	export COVERAGE_TARGETS
endif

.PHONY: all clean deps-update update-labels install-metrics-binaries lint periodic-jobs-gantt periodic-jobs-spread periodic-jobs-spread-dry-run $(limiter) $(flake-report-writer) $(querier) $(kubevirtci) $(flake-issue-creator)
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
	go build ./external-plugins/... ./releng/... ./robots/... ./github/ci/services/... ./cmd/... ./pkg/...

test: build
	go test ./external-plugins/... ./releng/... ./robots/... ./cmd/... ./pkg/...

update-labels:
	./hack/labels/update.sh

lint:
	./hack/lint.sh

coverage:
	if ! command -V covreport; then go install github.com/cancue/covreport@latest; fi
	go test ${COVERAGE_TARGETS} -coverprofile=/tmp/coverage.out
	mkdir -p ${ARTIFACTS}
	covreport  -i /tmp/coverage.out -o ${COVERAGE_OUTPUT_PATH}

periodic-jobs-gantt:
	@echo "Generating Gantt chart for periodic kubevirt/kubevirt e2e jobs..."
	go run ./cmd/periodic-jobs gantt

periodic-jobs-spread:
	@echo "Spreading periodic kubevirt/kubevirt e2e jobs..."
	go run ./cmd/periodic-jobs spread --verbose

periodic-jobs-spread-dry-run:
	@echo "Dry run: spreading periodic kubevirt/kubevirt e2e jobs..."
	go run ./cmd/periodic-jobs spread --dry-run --verbose

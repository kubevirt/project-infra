limiter     := limiter
flake-report-writer := robots/cmd/flake-report-writer
flake-issue-creator := robots/cmd/flake-issue-creator
querier := robots/cmd/release-querier
kubevirtci := robots/cmd/kubevirtci-bumper
bazelbin := bazelisk

ifndef GOLANGCI_LINT_VERSION
	GOLANGCI_LINT_VERSION=v1.62.2
	export GOLANGCI_LINT_VERSION
endif
ifndef ARTIFACTS
	ARTIFACTS=/tmp/artifacts
	export ARTIFACTS
endif
ifndef COVERAGE_OUTPUT_PATH
	COVERAGE_OUTPUT_PATH=${ARTIFACTS}/coverage.html
	export COVERAGE_OUTPUT_PATH
endif

.PHONY: all clean deps-update gazelle-update-repos update-labels install-metrics-binaries lint $(limiter) $(flake-report-writer) $(querier) $(kubevirtci) $(flake-issue-creator)
all: deps-update $(limiter) $(flake-report-writer) $(querier) $(kubevirtci) $(flake-issue-creator)

clean: install-metrics-binaries
	$(bazelbin) clean
	golangci-lint cache clean

$(limiter) $(flake-report-writer) $(querier) $(kubevirtci) $(flake-issue-creator): deps-update
	$(MAKE) --directory=$@

bazel-build-all:
	$(bazelbin) build //limiter:go_default_library //releng/... //robots/... //github/ci/services/...

deps-update:
	export GO111MODULE=on
	go get ./...
	go mod tidy
	go mod vendor
	sed -i "s|^.*data = \[\"//prow/plugins:config-src\"\],||g" vendor/k8s.io/test-infra/pkg/genyaml/BUILD.bazel
	$(bazelbin) run //:gazelle

gazelle:
	$(bazelbin) run //:gazelle

gazelle-update-repos:
	$(bazelbin) run //:gazelle -- update-repos -from_file=go.mod

install-bazelisk:
	go get -u github.com/bazelbuild/bazelisk

test:
	$(bazelbin) build //external-plugins/... //releng/... //robots/... //github/ci/services/...
	$(bazelbin) test //external-plugins/... //releng/... //robots/...

update-labels:
	./hack/labels/update.sh

install-metrics-binaries:
	if ! command -V golangci-lint; then curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ${GOPATH}/bin ${GOLANGCI_LINT_VERSION} ; fi

lint: install-metrics-binaries
	./hack/lint.sh

coverage:
	if ! command -V covreport; then go install github.com/cancue/covreport@latest; fi
	go test \
		./external-plugins/... \
		./releng/... \
		./robots/... \
		-coverprofile=/tmp/coverage.out
	mkdir -p ${ARTIFACTS}
	covreport  -i /tmp/coverage.out -o ${COVERAGE_OUTPUT_PATH}
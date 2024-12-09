limiter     := limiter
flake-report-writer := robots/cmd/flake-report-writer
flake-issue-creator := robots/cmd/flake-issue-creator
querier := robots/cmd/release-querier
kubevirtci := robots/cmd/kubevirtci-bumper
bazelbin := bazelisk

.PHONY: all clean deps-update gazelle-update-repos update-labels $(limiter) $(flake-report-writer) $(querier) $(kubevirtci) $(flake-issue-creator)
all: deps-update $(limiter) $(flake-report-writer) $(querier) $(kubevirtci) $(flake-issue-creator)

clean:
	$(bazelbin) clean

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
	if ! command -V gocyclo; then go install github.com/fzipp/gocyclo/cmd/gocyclo@latest ; fi

go-metrics: install-metrics-binaries
	gocyclo -over 10 .

limiter     := limiter
flake-report-writer := robots/cmd/flake-report-writer
flake-issue-creator := robots/cmd/flake-issue-creator
querier := robots/cmd/release-querier
kubevirtci := robots/cmd/kubevirtci-bumper
bazelbin := bazelisk

.PHONY: all clean deps-update gazelle-update-repos $(limiter) $(flake-report-writer) $(querier) $(kubevirtci) $(flake-issue-creator)
all: deps-update $(limiter) $(flake-report-writer) $(querier) $(kubevirtci) $(flake-issue-creator)

clean:
	$(bazelbin) clean

$(limiter) $(flake-report-writer) $(querier) $(kubevirtci) $(flake-issue-creator): deps-update
	$(MAKE) --directory=$@

deps-update:
	export GO111MODULE=on
	go get ./...
	go mod tidy
	go mod vendor
	sed -i "s|^.*data = \[\"//prow/plugins:config-src\"\],||g" vendor/k8s.io/test-infra/pkg/genyaml/BUILD.bazel
	$(bazelbin) run //:gazelle

gazelle:
	bazel run //:gazelle -- robots/

gazelle-update-repos:
	bazel run //:gazelle -- update-repos -from_file=go.mod

install-bazelisk:
	go get -u github.com/bazelbuild/bazelisk

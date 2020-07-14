limiter     := limiter
flakefinder := robots/flakefinder
querier := robots/release-querier
kubevirtci := robots/kubevirtci-bumper
bazelbin := bazelisk

.PHONY: all clean deps-update $(limiter) $(flakefinder) $(querier) $(kubevirtci)
all: deps-update $(limiter) $(flakefinder) $(querier) $(kubevirtci)

clean:
	$(bazelbin) clean

$(limiter) $(flakefinder) $(querier) $(kubevirtci): deps-update
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

install-bazelisk:
	go get -u github.com/bazelbuild/bazelisk

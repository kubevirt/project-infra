limiter     := limiter
flakefinder := robots/flakefinder
querier := robots/release-querier
kubevirtci := robots/kubevirtci-bumper

.PHONY: all clean deps-update $(limiter) $(flakefinder) $(querier) $(kubevirtci)
all: deps-update $(limiter) $(flakefinder) $(querier) $(kubevirtci)

clean:
	bazel clean

$(limiter) $(flakefinder) $(querier) $(kubevirtci): deps-update
	$(MAKE) --directory=$@

deps-update:
	export GO111MODULE=on
	go get ./...
	go mod tidy
	go mod vendor
	bazel run //:gazelle

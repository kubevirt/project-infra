limiter     := limiter
flakefinder := robots/flakefinder
querier := robots/release-querier

.PHONY: all clean deps-update $(limiter) $(flakefinder) $(querier)
all: deps-update $(limiter) $(flakefinder) $(querier)

clean:
	bazel clean

$(limiter) $(flakefinder) $(querier): deps-update
	$(MAKE) --directory=$@

deps-update:
	go mod tidy
	go mod vendor
	bazel run //:gazelle

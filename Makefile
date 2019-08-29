limiter     := limiter
flakefinder := robots/flakefinder

.PHONY: all clean deps-update $(limiter) $(flakefinder)
all: deps-update $(limiter) $(flakefinder)

clean:
	bazel clean

$(limiter) $(flakefinder): deps-update
	$(MAKE) --directory=$@

deps-update:
	go mod tidy
	go mod vendor
	bazel run //:gazelle

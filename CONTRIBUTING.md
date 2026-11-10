# Contributing to kubevirt/project-infra

Welcome! As stated in the [README](README.md) this repository provides supporting code for the project infrastructure.

See [the KubeVirt contribution guide](https://github.com/kubevirt/kubevirt/blob/main/CONTRIBUTING.md) for general information about how to contribute.

## Prerequisites

Use the Go version and toolchain specified in [go.mod](go.mod). The lint target
requires a locally installed `golangci-lint` and uses [.golangci.yaml](.golangci.yaml).

## Development Commands

Run these targets from the repository root. The [Makefile](Makefile) defines
the package lists and command details.

- `make build`: Build the Go packages selected by the Makefile.
- `make test`: Run `make build`, then test the selected Go packages.
- `make lint`: Run `golangci-lint` through `hack/lint.sh`.
- `make coverage`: Run coverage tests and generate an HTML report, installing
  `covreport` if needed.
- `make deps-update`: Run `go get ./...`, `go mod tidy`, and `go mod vendor`.
- `make clean`: Clear the golangci-lint cache and Go build and module caches.

`make coverage` writes its HTML report to `/tmp/artifacts/coverage.html` by
default. Override `ARTIFACTS`, `COVERAGE_OUTPUT_PATH`, or `COVERAGE_TARGETS` as
needed; its intermediate coverage profile is written to `/tmp/coverage.out`.

`make deps-update` can change `go.mod`, `go.sum`, and `vendor/`; review those
changes before submitting them. The default `make` target also runs
`deps-update`, so use `make build` when you only want to build the selected
packages. `make clean` clears shared Go caches, not just repository build
artifacts, and subsequent builds may need to download modules again.

## Task-Specific Guides

- [CI image development and publishing](images/README.md)
- [Label configuration and synchronization](docs/labels-management.md)
- [Generated label listing and update procedure](docs/labels.md)

Do not edit the generated label listing directly; use `make update-labels`
after changing the label configuration. The [repository overview](README.md)
links additional component documentation.

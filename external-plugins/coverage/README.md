# Coverage External Plugin

Automatically runs Go unit test coverage on pull requests containing Go code changes.

## Overview

This Prow external plugin:
- Listens for GitHub pull request webhooks (opened/synchronize events)
- Detects when a PR contains Go file changes (*.go, go.mod, go.sum)
- Automatically creates and submits a coverage ProwJob
- Makes coverage artifacts browsable via Prow's Spyglass coverage lens
- Posts GitHub status check with link to coverage report

## How It Works

1. GitHub sends webhook when PR is opened/updated
2. Plugin checks if PR contains Go files in coverage directories
3. If yes, plugin generates a ProwJob to run `make coverage`
4. Prow executes the job, uploads artifacts to GCS
5. Developer can view coverage in Spyglass UI via GitHub status link

## Coverage Scope

Includes Go files in:
- `external-plugins/`
- `releng/`
- `robots/`

## Configuration

See deployment manifests in:
- `github/ci/prow-deploy/kustom/base/manifests/local/prow-coverage-*.yaml`

## Related

- Issue: https://github.com/kubevirt/project-infra/issues/4064
- Design doc: `/COVERAGE_PLUGIN_DESIGN.md`
- Based on rehearse plugin pattern

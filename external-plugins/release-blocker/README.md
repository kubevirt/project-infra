# release-blocker - external plugin for Prow

## Motivation

When managing releases across multiple branches, teams need a way to signal
that a specific issue or PR must be resolved before the next release is cut.
The release-blocker plugin provides this by allowing project members to mark
issues and PRs with branch-specific blocker labels via slash commands, and
enforces that these labels cannot be manually added or removed through GitHub.

The labels created by this plugin are consumed by the [release-tool](https://github.com/kubevirt/project-infra/tree/main/releng/release-tool) during the release process.
The release-tool queries GitHub for open issues and PRs carrying these labels and
will block RC promotions and tag creation until all blockers for the target branch are resolved.

## Overview

Release-blocker is a Prow external plugin that:
- Listens for GitHub issue comment events (`created`) to handle `/release-blocker` commands on both issues and PRs
- Listens for issue and pull request `labeled`/`unlabeled` events to prevent manual label manipulation
- Applies labels in the format `release-blocker/{branch-name}` (e.g., `release-blocker/release-3.9`)
- Validates that the target branch exists on GitHub before adding a blocker label
- Reverses any manual additions or removals of `release-blocker/*` labels and posts a comment directing users to use the slash command

## Usage

Comment on an issue or PR to manage release blockers:

```
# Mark as a release blocker for a specific branch
/release-blocker <branch-name>

# Remove the release blocker label for a specific branch
/release-blocker cancel <branch-name>
```

Alternative command syntaxes are also accepted: `/release-block`, `/releaseblock`, `/releaseblocker`.

## Authorisation

A release blocker label can be added or removed if the user is both:
- A **top-level approver** from the repository's `OWNERS` file
- A **member** of the GitHub organization


## Configuration

The plugin is registered in:
- `github/ci/prow-deploy/kustom/base/configs/current/plugins/plugins.yaml`

Deployment manifests:
- `github/ci/prow-deploy/kustom/base/manifests/local/release-blocker_deployment.yaml`
- `github/ci/prow-deploy/kustom/base/manifests/local/release-blocker_service.yaml`

## Limitations

- Only one `/release-blocker` command per comment — if multiple commands are present, none of them will be processed

## Development

Build the binary:

```bash
make build
```

Run tests:

```bash
make test
```

Format, test, and push the image:

```bash
make all
```

Push the image only:

```bash
make push
```

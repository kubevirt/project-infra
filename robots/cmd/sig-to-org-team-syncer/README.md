# sig-to-org-team-syncer

Uses an `OWNERS_ALIASES` file as input to generate associated GitHub teams and GitHub labels with the respective configuration files.

## Motivation

There's no way to ping a group of people that are associated to a SIG on a pull request. But there's the ability to ping a GitHub team on a pull request. Thus this tool generates a GitHub team so that we can ping the group of people on a PR.

Also since it is frequently forgotten that labels associated to OWNERS filters need to be added to the label_sync configuration file, this tool adds the respective labels if they do not exist.

## How to run

```bash
go run robots/cmd/sig-to-org-team-syncer/main.go \
    --owners-aliases-path $(cd ../kubevirt && pwd)/OWNERS_ALIASES \
    --orgs-yaml-path=github/ci/prow-deploy/kustom/base/configs/current/orgs/orgs.yaml \
    --target-org-name=kubevirt \
    --labels-yaml-path=github/ci/prow-deploy/kustom/base/configs/current/labels/labels.yaml
```

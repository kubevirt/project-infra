# testutils - shared test library for external plugins

## Motivation

External Prow plugins need to test their GitHub and OWNERS file interactions
without making real API calls. Rather than each plugin maintaining its own
mocks, testutils provides a shared set of fake implementations that keep
test setup consistent and reduce duplication across the external-plugins
directory.

## Overview

testutils is a Go library package that provides three in-memory mock types:

- **`FakeClient`** — implements the Prow GitHub client interface with
  methods covering issues, PRs, comments, reviews, labels, statuses, refs,
  commits, teams, projects, milestones, reactions, and collaborators. All
  mutations are recorded (e.g. `IssueLabelsAdded`, `IssueCommentsDeleted`,
  `AssigneesAdded`) so tests can assert on exactly what the plugin did.

- **`FakeOwnersClient`** — implements the Prow `RepoOwner` interface for
  testing OWNERS file workflows: top-level approvers, leaf approvers,
  reviewers, required reviewers, and OWNERS config parsing.

- **`FakeRepoownersClient`** — thin wrapper that satisfies the
  `RepoOwnerLoader` interface by returning a `FakeOwnersClient`.

## Usage

Import the package and initialise the fakes with the state your test needs:

```go
import "kubevirt.io/project-infra/external-plugins/testutils"

fc := &testutils.FakeClient{
    PullRequests: map[int]*github.PullRequest{ /* ... */ },
    IssueLabelsExisting: []string{ /* ... */ },
}

foc := &testutils.FakeOwnersClient{
    ExistingTopLevelApprovers: sets.New[string]("approver-a"),
}
froc := testutils.FakeRepoownersClient{Foc: foc}
```

Currently used by:
- [rehearse](../rehearse/) — handler and plugin tests
- [release-blocker](../release-blocker/) — server tests

## Limitations

- `FakeOwnersClient` has not been fully implemented, `RepoOwner` interface
  `AllOwners()`, `AllApprovers()`, and `AllReviewers()` will panic if called
- Several `FakeClient` methods return hardcoded fixtures rather than using
  configurable fields: `GetRepos` always returns `kubernetes/kubernetes` and
  `kubernetes/community`, `ListTeams` returns two fixed teams (Admins ID 0,
  Leads ID 42), `ListTeamMembers` has hardcoded members, and `AssignIssue`
  triggers `MissingUsers` error when the string `"not-in-the-org"` is
  assigned.

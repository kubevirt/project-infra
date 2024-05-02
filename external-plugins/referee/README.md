# referee - external plugin for prow

Referee is an external plugin for [prow] that helps ci maintainers with enforcing specific rules that we have set to keep KubeVirt CI healthy.

Currently it does the following:

* place a `/hold` on pull requests that exceed a certain number of retests after the last change

## Motivation

We have noticed that at certain times pull requests are constantly retested without any chance of succeeding. Our reasoning is that if a pull request is retested a certain number of times without any changes this points to an instability or flakiness that people need to look at and fix.

## Metrics

### Retests

> [!NOTE]
> counters are reset after plugin restart

Below is a copy of the generated metrics for retests with explanation:

	# HELP referee_retests_org_repo_total The total number of retests for org_repo_total encountered so far
	# TYPE referee_retests_org_repo_total counter
	referee_retests_org_repo_total 1
	# HELP referee_retests_total The total number of retests encountered so far
	# TYPE referee_retests_total counter
	referee_retests_total 1
	# HELP referee_retests_org_repo_pr_since_last_commit The number of retests per PR since last commit in org/repo encountered so far
	# TYPE referee_retests_org_repo_pr_since_last_commit gauge
	referee_retests_org_repo_pr_since_last_commit{pull_request="1742"} 37

[prow]: prow.ci.kubevirt.io

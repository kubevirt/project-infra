# robots/cmd/kubevirt

Provides commands to manipulate the SIG testing job definitions that execute the functional tests for kubevirt/kubevirt.

## Commands provided

### `kubevirt check providers`

`kubevirt check providers` checks the usage of kubevirtci providers for periodic and presubmit jobs for kubevirt/kubevirt

For each of the periodics and presubmits for kubevirt/kubevirt it matches the TARGET env variable value of all containers specs against the provider name pattern and records all matches.
It then generates a list of used providers, separated in unsupported and supported ones.

```
Usage:
kubevirt check providers [flags]

Flags:
--fail-on-unsupported-provider-usage           Whether to exit with non zero exit code in case an unsupported provider usage is detected (default true)
-h, --help                                         help for providers
--job-config-path-kubevirt-periodics string    The path to the kubevirt periodic job definitions
--job-config-path-kubevirt-presubmits string   The path to the kubevirt presubmit job definitions
--output-file string                           Path to output file, if not given, a temporary file will be used
--overwrite                                    Whether to overwrite output file

Global Flags:
--dry-run                    Whether the file should get modified or just modifications printed to stdout. (default true)
--github-endpoint string     GitHub's API endpoint (may differ for enterprise). (default "https://api.github.com/")
--github-token-path string   Path to the file containing the GitHub OAuth secret. (default "/etc/github/oauth")
```

### `kubevirt copy jobs`

`kubevirt copy jobs` creates copies of the periodic and presubmit SIG jobs for latest kubevirtci providers

For each of the sigs (sig-network, sig-storage, sig-compute, sig-operator)
it checks whether a job to run with the latest k8s version exists.
If not, it copies the existing job for the previously latest kubevirtci provider and
adjusts it to run with an eventually soon-to-be-existing new kubevirtci provider.

Presubmit jobs will be created with

        always_run: false
        optional: true

to avoid them failing all the time until the new provider is integrated into kubevirt/kubevirt.

```
Usage:
kubevirt copy jobs [flags]

Flags:
-h, --help                                         help for jobs
--job-config-path-kubevirt-periodics string    The path to the kubevirt periodic job definitions
--job-config-path-kubevirt-presubmits string   The path to the kubevirt presubmit job definitions
--k8s-release-semver string                    The semver of the k8s release to create the jobs for, or (as default) empty string to create for latest release

Global Flags:
--dry-run                    Whether the file should get modified or just modifications printed to stdout. (default true)
--github-endpoint string     GitHub's API endpoint (may differ for enterprise). (default "https://api.github.com/")
--github-token-path string   Path to the file containing the GitHub OAuth secret. (default "/etc/github/oauth")
```

### `kubevirt get periodics`

`kubevirt get periodics` describes periodic job definitions in project-infra for kubevirt/kubevirt repo

It reads the job configurations for kubevirt/kubevirt e2e periodic jobs, extracts information and creates a table in
html format, so that we can quickly see which job is running how often and when. Also the table is sorted in running
order, where jobs that run more often are ranked higher in the list.

```
Usage:
kubevirt get periodics [flags]

Flags:
-h, --help                                        help for periodics
--job-config-path-kubevirt-periodics string   The path to the kubevirt periodic job definitions
--output-file string                          The file to write the output to, if empty, a temp file will be generated. If file exits, it will be overwritten

Global Flags:
--dry-run                    Whether the file should get modified or just modifications printed to stdout. (default true)
--github-endpoint string     GitHub's API endpoint (may differ for enterprise). (default "https://api.github.com/")
--github-token-path string   Path to the file containing the GitHub OAuth secret. (default "/etc/github/oauth")
```

### `kubevirt get presubmits` 

`kubevirt get presubmits` describes presubmit job definitions in project-infra for kubevirt/kubevirt repo

It reads the job configurations for kubevirt/kubevirt e2e presubmit jobs, extracts information and creates a table in html
format, so that we can quickly see which job is gating the merge and which job is running on every kubevirt/kubevirt PR.

The table is sorted in order gating -> always_run -> conditional_runs -> others and can be filtered by job name.

```
Usage:
kubevirt get presubmits [flags]

Flags:
-h, --help                                         help for presubmits
--job-config-path-kubevirt-presubmits string   The path to the kubevirt presubmits job definitions
--output-file string                           The file to write the output to, if empty, a temp file will be generated. If file exits, it will be overwritten

Global Flags:
--dry-run                    Whether the file should get modified or just modifications printed to stdout. (default true)
--github-endpoint string     GitHub's API endpoint (may differ for enterprise). (default "https://api.github.com/")
--github-token-path string   Path to the file containing the GitHub OAuth secret. (default "/etc/github/oauth")
```

### `kubevirt remove always_run`

`kubevirt remove always_run` sets always_run to false on presubmit job definitions for kubevirt for unsupported kubevirtci providers

For each of the sigs (sig-network, sig-storage, sig-compute, sig-operator)
it sets always_run to false on presubmit job definitions that contain
"old" k8s versions. From kubevirt standpoint an old k8s version is one
that is older than one of the three minor versions including the
currently released k8s version at the time of the check.

Example:

* k8s 1.22 is the current stable version
* job definitions exist for k8s 1.22, 1.21, 1.20
* presubmits for 1.22 are to run always and are required (aka optional: false)
* job definitions exist for 1.19 k8s version

This will lead to always_run being set to false for each of the sigs presubmit jobs for 1.19

See kubevirt k8s version compatibility: https://github.com/kubevirt/kubevirt/blob/main/docs/kubernetes-compatibility.md#kubernetes-version-compatibility

```
Usage:
kubevirt remove always_run [flags]

Flags:
--force                                        skip check of always_run, disable right away
-h, --help                                         help for always_run
--job-config-path-kubevirt-presubmits string   The directory of the kubevirt presubmit job definitions

Global Flags:
--dry-run                    Whether the file should get modified or just modifications printed to stdout. (default true)
--github-endpoint string     GitHub's API endpoint (may differ for enterprise). (default "https://api.github.com/")
--github-token-path string   Path to the file containing the GitHub OAuth secret. (default "/etc/github/oauth")
```

### `kubevirt remove jobs`

`kubevirt remove jobs` removes presubmit and periodic job definitions for kubevirt for unsupported kubevirtci providers

For each of the sigs (sig-network, sig-storage, sig-compute, sig-operator)
it removes job definitions that contain "old"
k8s versions. From kubevirt standpoint an old k8s version is one
that is older than one of the three minor versions including the
currently released k8s version at the time of the check.

Example:

* k8s 1.22 is the current stable version
* job definitions exist for k8s 1.22, 1.21, 1.20
* presubmits for 1.22 are to run always and are required (aka optional: false)
* job definitions exist for 1.19 k8s version

This will lead to each of the sigs periodic and presubmit jobs for 1.19 being removed

See kubevirt k8s version compatibility: https://github.com/kubevirt/kubevirt/blob/main/docs/kubernetes-compatibility.md#kubernetes-version-compatibility

```
Usage:
kubevirt remove jobs [flags]

Flags:
--force                                        Whether the job definitions should be removed regardless of the state
-h, --help                                         help for jobs
--job-config-path-kubevirt-periodics string    The path to the kubevirt periodic job definitions
--job-config-path-kubevirt-presubmits string   The directory of the kubevirt presubmit job definitions

Global Flags:
--dry-run                    Whether the file should get modified or just modifications printed to stdout. (default true)
--github-endpoint string     GitHub's API endpoint (may differ for enterprise). (default "https://api.github.com/")
--github-token-path string   Path to the file containing the GitHub OAuth secret. (default "/etc/github/oauth")
```

### `kubevirt require presubmits` 

`kubevirt require presubmits` moves presubmit job definitions for kubevirt to being required to merge

For each of the sigs (sig-network, sig-storage, sig-compute, sig-operator)
it aims to make the jobs for the latest kubevirtci provider
required and run on every PR. This is done in two stages.
First it sets for a job that doesn't always run the

        always_run: true
        optional: false

flag. This will make the job run on every PR but failing checks
will not block the merge.

On second stage, it removes the

        optional: false

which makes the job required to pass for merges to occur with tide.

```
Usage:
kubevirt require presubmits [flags]

Flags:
-h, --help                                         help for presubmits
--job-config-path-kubevirt-presubmits string   The directory of the kubevirt presubmit job definitions

Global Flags:
--dry-run                    Whether the file should get modified or just modifications printed to stdout. (default true)
--github-endpoint string     GitHub's API endpoint (may differ for enterprise). (default "https://api.github.com/")
--github-token-path string   Path to the file containing the GitHub OAuth secret. (default "/etc/github/oauth")
```

Building
--------

    make all

will build, test and check the project

See Makefile for details.

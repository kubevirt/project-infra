# test-report

`test-report` creates a report about which tests have been run (or not run) on what lane. Especially it considers the `dont_run_tests.json` files per branch of the lanes that are considered.


## Provided default configurations

There are four default configurations, where `default` considers the lanes for all sigs, while `compute`, `network`, `ssp` and `storage` only consider those tests and lanes that are relevant to each respective sig. For details on the configuration of each use case, see the config files in this folder.

If you want to provide your own configuration, you can override the default configs using `--config-file` flag.


## Running the report


### using shell wrapper script with podman

The easiest way for running the report is using the shell script. You can find it [here](../../../hack/test-report.sh)

**Note: Since the app is containerized, [`podman`](https://podman.io/) is required for the script!** 

```shell
$ ./hack/test-report.sh --help
test-report creates a report about which tests have been run on what lane

Usage:
test-report [flags]

Flags:
--config string         one of {'default', 'compute', 'storage', 'network', 'ssp'}, chooses one of the default
                        configurations, if set overrides default-config.yaml (default "default")
--config-file string    yaml file that contains job names associated with dont_run_tests.json and the job name pattern,
                        if set overrides default-config.yaml
--dry-run               only check which jobs would be considered, do not create an actual report
--endpoint string       jenkins base url (default "https://main-jenkins-csb-cnvqe.apps.ocp-c1.prod.psi.redhat.com/")
-h, --help              help for test-report
--log-level uint32      level for logging (default 4)
--outputFile string     Path to output file, if not given, a temporary file will be used
--overwrite             overwrite output file (default true)
--start-from duration   time period for report (default 336h0m0s)
```


### using golang

You can also use [`go`](https://go.dev/) to run it:

```shell
$ go run ./robots/cmd/test-report/... --help
test-report creates a report about which tests have been run on what lane
...
```


### using bazel

You can use [`bazel`]() to run it:

```shell
$ bazel run //robots/cmd/test-report -- --help
INFO: Analyzed target //robots/cmd/test-report:test-report (0 packages loaded, 0 targets configured).
...
test-report creates a report about which tests have been run on what lane
...
```


## Checking custom configuration

Use ``--dry-run`` flag to see which jobs are considered for the report ahead of actually creating the report.

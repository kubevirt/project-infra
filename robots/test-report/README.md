# test-report

Creates reports over test results in connection with `quarantined_tests.json` and `dont_run_tests.json` file types.

## test-report execution

test-report execution creates a report in html format to show which tests have been run on what lane

It constructs a matrix of test lanes by test names and shows for each test:
* on which lane(s) the test actually was run
* on which lane(s) the test is not supported (taking the information from the 'dont_run_tests.json' configured for the lane
  (see the configurations available)

Tests that are not run on any lane are especially marked in order to emphasize that fact.

Accompanying the html file a json data file is emitted for further consumption.

**Note**: generating a default report can take a while and will emit a report of enormous size, therefore you can strip down
the output by selecting a configuration that reports over a subset of the data using --config. You also can create your
own configuration file to adjust the report output to your requirements.

```
Usage:
test-report execution [flags]

Flags:
--config string         one of {'default', 'compute', 'storage', 'network'}, chooses one of the default configurations, if set overrides default-config.yaml (default "default")
--config-file string    yaml file that contains job names associated with dont_run_tests.json and the job name pattern, if set overrides default-config.yaml
--dry-run               only check which jobs would be considered, do not create an actual report
--endpoint string       jenkins base url (default "https://main-jenkins-csb-cnvqe.apps.ocp-c1.prod.psi.redhat.com/")
-h, --help                  help for execution
--outputFile string     Path to output file, if not given, a temporary file will be used
--overwrite             overwrite output file (default true)
--start-from duration   time period for report (default 336h0m0s)

Global Flags:
--log-level uint32   level for logging (default 4)
```

### Provided default configurations

There are four default configurations, where `default` considers the lanes for all sigs, while `compute`, `network`, `ssp` and `storage` only consider those tests and lanes that are relevant to each respective sig. For details on the configuration of each use case, see the config files in this folder.

If you want to provide your own configuration, you can override the default configs using `--config-file` flag.

## test-report dequarantine report

`test-report dequarantine report` generates a report of the test status for each entry in the `quarantined_tests.json`

The output format is an extended version of the format from `quarantined_tests.json`, added to each record is a
dictionary of test results per test that matches 'Id', ordered by execution time descending.

```
Usage:
  test-report dequarantine report [flags]

Flags:
      --dry-run                      whether to only check what jobs are being considered and then exit (default true)
      --endpoint string              jenkins base url (default "https://main-jenkins-csb-cnvqe.apps.ocp-c1.prod.psi.redhat.com/")
  -h, --help                         help for report
      --job-name-pattern string      the pattern to which all jobs have to match
      --max-conns-per-host int       the maximum number of connections that are going to be made (default 3)
      --output-file string           Path to output file, if not given, a temporary file will be used
      --quarantine-file-url string   the url to the quarantine file
      --start-from duration          time period for report (default 240h0m0s)

Global Flags:
      --log-level uint32   level for logging (default 4)
```

## test-report dequarantine execute

`test-report dequarantine execute` creates a new file matching the format of quarantined_tests.json from the source file where entries for stable tests are omitted

to do that the Jenkins server is asked for build results from the matching lanes within the matching time frame,
then results are filtered by those tests whose names match the entries in the `quarantined_tests.json`.

The remaining build results are inspected for failures. If one of the following conditions applies

* any failure is seen
* not reaching a minimum amount of passed tests (see `--minimum-passed-runs-per-test`)

then that test is seen as unstable and the entry will be transferred into the new file.

The execution filtering logs all activity to make clear why a test is not considered as stable. All output regarding 
a test not being considered as stable is done on warning level.

Output on error level is emitted if the record under inspection can not be matched to any test from the test results
that have been acquired.

```shell
Usage:
  test-report dequarantine execute [flags]

Flags:
      --dry-run                            whether to only check what jobs are being considered and then exit (default true)
      --endpoint string                    jenkins base url (default "https://main-jenkins-csb-cnvqe.apps.ocp-c1.prod.psi.redhat.com/")
  -h, --help                               help for execute
      --job-name-pattern string            the pattern to which all jobs have to match
      --max-conns-per-host int             the maximum number of connections that are going to be made (default 3)
      --minimum-passed-runs-per-test int   whether to only check what jobs are being considered and then exit (default 2)
      --output-file string                 Path to output file, if not given, a temporary file will be used
      --quarantine-file-url string         the url to the quarantine file
      --start-from duration                time period for report (default 240h0m0s)

Global Flags:
      --log-level uint32   level for logging (default 4)

```

## Running test-report commands


### using shell wrapper script with podman

The easiest way for running `test-report` is using the shell script. You can find it [here](../../hack/test-report.sh)

**Note: Since the app is containerized, [`podman`](https://podman.io/) is required for the script!** 

```shell
$ ./hack/test-report.sh --help
test-report creates a report about which tests have been run on what lane

Usage:
...
```


### using golang

You can also use [`go`](https://go.dev/) to run it:

```shell
$ go run ./robots/cmd/test-report/... --help
test-report creates a report about which tests have been run on what lane
...
```


## Checking custom configuration

Use ``--dry-run`` flag to see which jobs are considered for the report ahead of actually creating the report.

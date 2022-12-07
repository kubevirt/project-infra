# test-report

`test-report` creates a report about which tests have been run (or not run) on what lane. Especially it considers the `dont_run_tests.json` files per branch of the lanes that are considered.

```
Usage:
test-report [flags]

Flags:
--config string         one of {'default', 'compute', 'storage', 'network'}, chooses one of the default configurations, 
                        if set overrides default-config.yaml (default "default")
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

## Provided default configurations

There are four default configurations, where `default` considers the lanes for all sigs, while `compute`, `network` and `storage` only consider those tests and lanes that are relevant to each respective sig. For details on the configuration of each use case, see the [configs folder](./configs/).

If you want to provide your own configuration, you can override the default configs using `--config-file` flag.

## Checking provided configuration

Use ``--dry-run`` flag to see which jobs are considered for the report ahead of actually creating the report.

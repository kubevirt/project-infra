# per-test-execution

Creates a set of CSV files in the format

| test-name       | number-of-executions     | number-of-failures     | {build-url-1} | ... | {build-url-m} |
|-----------------|--------------------------|------------------------|---------------|-----|---------------|
| {{test-name-1}} | {{number-of-executions}} | {{number-of-failures}} | {p,f,s}       | ... | {p,f,s}       |
| ...             | ...                      | ...                    | ...           | ... | ...           |
| {{test-name-n}} | {{number-of-executions}} | {{number-of-failures}} | {p,f,s}       | ... | {p,f,s}       |

where `n` is the number of tests and `m` is the number of builds.

Data is fetched from the build artifacts of the prow jobs, namely the `junit.xml` files that contain the test result data.

In detail the process works as follows:
* Per a target directory depending on the job we want to look at, we look at it's content directories, which resemble a job run each
* each job run we check whether it's inside the interval we want, and then extract the junit results
* we then aggregate the results and write them into a csv file

## Pre-requisites

env `GOOGLE_APPLICATION_CREDENTIALS` set to point to a GCS credentials file - see GCloud credentials docs [1] .

## Examples

```bash
go run ./robots/cmd/per-test-execution --days 7
```

```bash
go run ./robots/cmd/per-test-execution --months 6
```

[1]: https://cloud.google.com/docs/authentication/application-default-credentials
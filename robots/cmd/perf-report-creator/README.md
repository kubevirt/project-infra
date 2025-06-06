#### Perf Report Creator

This tool is used to process the sig-scale CI jobs and extract performance numbers from each job. This tool can be used
to create weekly graphs of the performance metrics in CI and upload it in a different git repository.
The tools works in three steps:

1. results
2. weekly-report
3. weekly-graph

##### Step 1:

###### Usage

```shell
$ perf-report-creator results --help
Usage of results:
  -credentials-file string
        the credentials json file for GCS storage client
  -output-dir string
        the output directory were json data will be written (default "output/results")
  -performance-job-name string
        usuage, name of the performance job for which data is collected (default "periodic-kubevirt-e2e-k8s-1.25-sig-performance")
  -since duration
        Filter the periodic job in the time window (default 24h0m0s)
```

This step uses regex to match certain strings and grab results from build-log.txt. It organizes the output dir as 
`<output-dir>/<job-name>/<job-id>/data/results.json`

##### Step 2:

###### Usage

```shell
$ perf-report-creator weekly-report --help
Usage of weekly-report:
  -output-dir string
        the output directory were json data will be written (default "output/weekly")
  -results-dir string
        usuage, name of the performance job for which data is collected (default "output/results/periodic-kubevirt-e2e-k8s-1.25-sig-performance")
  -since duration
        Filter the periodic job in the time window (default 24h0m0s)
  -vm-metrics-list string
        comma separated list of metrics to be extracted for vms (default "vmiCreationToRunningSecondsP95")
  -vmi-metrics-list string
        comma separated list of metrics to be extracted for vmis (default "vmiCreationToRunningSecondsP95")

```

This step uses the data from step 1 and organizes the output based on the resource and metric in weekly batches. The 
output directory format is `<output-dir>/<vmi/vm>/<week-start-date>/data/results.json`


##### Step 3:

###### Usage

```shell
$ perf-report-creator weekly-graph --help
Usage of weekly-graph:
  -metrics-list string
        comma separated list of metrics to be plotted (default "vmiCreationToRunningSecondsP95")
  -plotly-html
        boolean for selecting what kind of graph will be plotted (default true)
  -is-release
        boolean for selecting if the graph is for a release version (default false)
  -resource string
        resource for which the graph will be plotted (default "vmi")
  -weekly-reports-dir string
        the output directory from which weekly json data will be read (default "output/weekly")

```

This step uses the data from step 2 and creates a graph based on the input from step 2. It creates `<output-dir>/<vmi/vm>/<week-start-date>/plot.png`
or `<output-dir>/<vmi/vm>/<week-start-date>/index.html` depending on the command line flags.

##### Overall Usage

All the three steps will be run in sequence to generate weekly graphs that can be uploaded to a separate git repository
to keep track of performance numbers over time/versions.

#### Performance Benchmark Release Graphs

This describes how performance benchmark graphs are generated for KubeVirt releases.

##### Overview

The scripts in this directory create release graphs that track KubeVirt performance metrics over time. The graphs are generated from benchmark data collected during periodic test runs.

##### DEVELOPERS GUIDE

Before running the graph generation scripts, ensure you have:
1. Update the 
    - `RELEASE_VERSION` - Version to generate graphs for (in format, e.g. "v1-6") 
    - `SINCE_DATE` - Start date to collect data from (in "YYYY-MM-DD" format, e.g. "2024-01-01")
     
     for a `post-project-infra-kubevirt-releasegraph` prow job in `/Users/svarnam/Git/code_repos/project-infra/github/ci/prow-deploy/files/jobs/kubevirt/project-infra/project-infra-postsubmits.yaml`

2. Update the `shape.yaml` file with all the fields for the release version. and remove the data for the releases before the since date.

     Example:
     ```
     - type: line
       x0: "2024-03-25"
       x1: "2024-03-25"
       y0: 0
       y1: 1
       yref: paper
       editable: true
       line:
         color: blue
         width: 2
         dash: dot
       label:
         text: k8s-1.29
         xanchor: right
     ```

As it is a postsubmit job it is triggered when it sees any change in the `/Users/svarnam/Git/code_repos/project-infra/robots/cmd/perf-report-creator/shape.yaml` file.

This will create a new directory in the ci-performance-benchmarks repo with the name as the latest version(eg. v1.6.0) which has the plots for the given release version. Commit and push the changes to the remote repository and create a PR using the credentials provided in the environment variables token, name and email.

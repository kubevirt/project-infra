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

This section describes how to generate performance benchmark graphs for KubeVirt releases.

##### Overview

The release graph generation process creates visual representations of KubeVirt performance metrics over time, specifically for release versions. These graphs help track performance trends and identify any regressions or improvements between releases.

##### DEVELOPERS GUIDE

1. Update the `shape.yaml` file:
   - Add the new release version and since date
   - Add line shapes for marking important dates (releases, k8s version changes)
   - Remove data for releases before the since date
   - Example configuration:
     ```yaml
     ReleaseVersion: "v1-6"
     SinceDate: "2024-01-01"
     LineShapes:
       - Type: "line"
         X0: "2024-03-25"
         X1: "2024-03-25"
         Y0: 0
         Y1: 1
         Yref: "paper"
         Editable: true
         Line:
           Color: "blue"
           Width: 2
           Dash: "dot"
         Label:
           Text: "k8s-1.29"
           Xanchor: "right"
     ```

##### Workflow

1. The process is triggered automatically when changes are made to the `shape.yaml` file as part of prow job `post-project-infra-kubevirt-releasegraph`
2. The job will:
   - Collect performance data since the specified date
   - Generate graphs with release markers
   - Create a new directory in the `ci-performance-benchmarks` repository named after the release version (e.g., `v1-6`)
   - Create a PR with the generated graphs

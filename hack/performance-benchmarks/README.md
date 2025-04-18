# Performance Benchmark Release Graphs

This document describes how to generate performance benchmark graphs for KubeVirt releases.

## Overview

The scripts in this directory create release graphs that track KubeVirt performance metrics over time. The graphs are generated from benchmark data collected during periodic test runs.

## Prerequisites

Before running the graph generation scripts, ensure you have:

1. Docker or Podman container runtime installed and running
2. A fork of the `ci-performance-benchmarks` repository
3. GitHub account with write access to your forked repository
4. GitHub personal access token exported as `GITHUB_TOKEN` environment variable
5. The following environment variables set:
   - `GIT_AUTHOR_NAME` - Your GitHub username
   - `GIT_AUTHOR_EMAIL` - Your GitHub email
   - `RELEASE_VERSION` - Version to generate graphs for (e.g. "v1-0") 
   - `SINCE_DATE` - Start date to collect data from (e.g. "2024-01-01")
6. Update the shape.yaml file with all the fields for the release version. and remove the data for the releases before the since date.
Example:
```
- type: line
  x0: "2024-12-05"  # date of the release
  x1: "2024-12-05"  # date of the release
  y0: 0
  y1: 1
  yref: paper
  editable: true  
  line:
    color: violet # color of the line
    width: 2
    dash: dot
  label:
    text: k8s-1.31   # release version/k8s version
    xanchor: right
```

Run the following command to plot the release graph
```
  make plot-release-graph
```

This will create a new directory in the ci-performance-benchmarks repo with the name as the latest version(eg. v1.6.0) which has the plots for the given release version. Commit and push the changes to the remote repository and create a PR using the credentials provided in the environment variables token, name and email.
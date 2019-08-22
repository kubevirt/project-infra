flakefinder
===========

`flakefinder` is a tool to find flaky tests.

`flakefinder` does the following:

1. Report creation
  i. Fetching all merged PRs within a given time period
  ii. Getting the last commit of each PR which got merged
  iii. Correlate the PR with all prowjobs which were running against this last commit
2. creates/updates html document in GCS bucket /reports/flakefinder/
  i. flakefinder-$date.html - extract skipped/failed/success from the junit results and create a html table
  ii. index.html

Selecting the right builds:

- Filter out not merged PRs
- Filter out identical prow jobs on multiple PRs (can be because of the merge pool)
- Filter out jobs which don't have a junit result
- Only shows test results for all lanes where a test at least failed once on one of the found lanes
- Only take prow jobs into account which were run on the commit which got merged

How to update flakefinder
-------------------------

### Build

    bazel run //robots/flakefinder:app


### Push the flakefinder image

    bazel run //robots/flakefinder:push --host_force_python=PY2


### Test flakefinder job locally

Prerequisites:
- a github personal access token in a file `oauth` for the target repo and 
- service account credentials in a file `service-account.json`.

#### Create a job definition using [`mkpj`](https://github.com/kubernetes/test-infra/tree/master/prow/cmd/mkpj)

    mkpj --config-path config.yaml --job-config-path jobs/kubevirt/kubevirt-periodics.yaml --job periodic-publish-flakefinder-reports > /tmp/prowjob.yaml

#### Run the job locally using [`phaino`](https://github.com/kubernetes/test-infra/tree/master/prow/cmd/phaino) 

    phaino --privileged /tmp/prowjob.yaml
    INFO[0000] Reading...                                    path=/tmp/prowjob.yaml
    INFO[0000] Converting job into docker run command...     job=periodic-publish-flakefinder-reports
    local /etc/github path ("token" mount): /home/dhiller/.tokens/etc/github
    local /etc/gcs path ("gcs" mount): /home/dhiller/.gcs/credentials
    "docker" "run" "--rm=true" \
     "--name=phaino-24602-1" \
     "--entrypoint=/app/robots/flakefinder/app.binary" \                        
     "-e" \                                                                     
     "GOOGLE_APPLICATION_CREDENTIALS=/etc/gcs/service-account.json" \
     "-v" \
     "/home/dhiller/.tokens/etc/github:/etc/github" \
     ...
    INFO[0007] Starting job...                               job=periodic-publish-flakefinder-reports                                                                                             
    INFO[0007] Waiting for job to finish...                  container=phaino-24602-1 job=periodic-publish-flakefinder-reports                                                                    
    Unable to find image 'kubevirtci/flakefinder@sha256:491cb61028bd5fcb3ae5ae3e79497f7cf3b4d4e68b70b7471cdc7359a3123e86' locally                                                                 
    sha256:491cb61028bd5fcb3ae5ae3e79497f7cf3b4d4e68b70b7471cdc7359a3123e86: Pulling from kubevirtci/flakefinder                                                                                  
    ...
    2019/08/22 14:43:59 report.go:188: Report will be written to gs://kubevirt-prow/reports/flakefinder/flakefinder-2019-08-22.html                                                               
    2019/08/22 14:44:00 index.go:68: Report index page will be written to gs://kubevirt-prow/reports/flakefinder/index.html                                                                       
    INFO[0145] PASS                                          duration=2m25.918744963s job=periodic-publish-flakefinder-reports                                                                    
    INFO[0145] SUCCESS                                                                                                                                                                            

#### Update job configuration to use new image

Update [`kubevirt-periodics.yaml`](github/ci/prow/files/jobs/kubevirt/kubevirt-periodics.yaml) with the new image sha.

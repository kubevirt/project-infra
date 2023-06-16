# KubeVirt Prow


[KubeVirt Prow] is the dedicated [Prow](https://docs.prow.k8s.io/docs/overview/) instance for the KubeVirt organization.

[KubeVirt Prow] is the landing page for our Prow instance. This is the place where the status of all the jobs run on it can be inspected. Furthermore it serves as an entry for status of prow components and help of plugins to look at.

One example is [Tide](https://docs.prow.k8s.io/docs/components/core/tide/), which takes care of automated merging of PRs (if configured for a repository).

The burger menu to the upper left holds links towards:
* _Prow status_ - the job overview page
* _PR status_ - a list of pull requests for a user and which status the are in currently
* _Command help_ - list of commands that are available for pull requests and docs
* _Tide status_ - current status, i.e. which PRs are currently tests, possibly as a batch, etc.
* _Tide history_ - which PR has been handled when and how
* _Plugins_ - list of installed plugins and docs
* _Documentation_ - the main entrance to prow documentation

KubeVirt GitHub organization management is done via [Peribolos](https://docs.prow.k8s.io/docs/components/cli-tools/peribolos/), which itself is run as an hourly periodic job on Prow.

See also: https://github.com/kubevirt/community/tree/main/docs

## Prow configuration

Configuration files are found in this folder: https://github.com/kubevirt/project-infra/tree/main/github/ci/prow-deploy/files

**Note: some of these need to be a symlink to the real files, please follow these and edit in target location.**

| Type         | Holds configuration for                    |
|--------------|--------------------------------------------|
| orgs.yaml    | KubeVirt GitHub organization               |
| config.yaml  | Prow components, prow job presets          |
| plugins.yaml | Prow plugins                               |
| labels.yaml  | GitHub labels for pull requests and issues |
| mirror.yaml  | image mirror                               |
| jobs         | job configurations per repository          |




## Prow and Prow Jobs

Prow runs all our test job configurations for the projects, i.e. when a PR on kubevirt/kubevirt is created or updated, [presubmit jobs configured here](https://github.com/kubevirt/project-infra/blob/main/github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-presubmits.yaml) are run. It also runs periodics (i.e. daily jobs) and postsubmits.
All Prow job configurations are hosted on [kubevirt/project-infra](https://github.com/kubevirt/project-infra). See [docs for job configuration](https://docs.prow.k8s.io/docs/jobs/).

![Prow - Deck](prow-deck.png)

This is the entry page provided by [Deck](https://docs.prow.k8s.io/docs/components/core/deck/) where you have an overview of all job runs on the Prow instance. From here you can filter on various criteria to select the jobs that you need to have a look at.

## Prow Job types

| Type | When run                                                                                              |
| ---- |-------------------------------------------------------------------------------------------------------|
| presubmit | after a commit is pushed to a pull request                                                            |
| postsubmit | after a pull request is merged to a branch                                                            |
| batch | whenever a batch run is initiated by Tide to test several pull requests before merge at the same time |
| periodic | on a defined interval                                                                                 |

## Logs and Artifacts on job details page

![Prow Job Details Page](prow-job-details.png)

After you click on a job link in the prow status page you are taken to the job run overview page that, when finished, contains information about tests (if available), the log file (normally showing the relevant portion to highlight where it failed), and links to details (i.e. full log (see at the bottom).

At the top of the page you see links to other detail sections:
* job history - shows you the history of all job runs on a presubmit job for a specific lane.
* PR history - shows you all test job runs in a grid. Column headers are the commit ids that have the jobs been run on. Row headers are the job names.
* artifacts - shows a browseable list of all files that have been captured for this test run. You can drill down here into the folder.

For jobs that have finished (either failed or succeeded) you can get further job information at the bottom of the page.

### Artifacts captured by k8s reporter

![Artifacts captured by k8s-reporter](prow-artifacts.png)

Each file in the artifacts folder has a numerical prefix. In this case we only see files prefixed with  1_. All files with the same numerical prefix belong to a specific test failure. Since only one test failed on this lane, only one prefix exists.The log files only  contain logs, yaml or events which happened during the runtime of this specific job. This is a big difference to must-gather which just dumps everything it sees (e.g. full log history), which is not suitable for debugging flaky tests.

**If you don't know where to start looking, open the `overview.log`, then go to the `events.log`.**

See also:
* [KubeVirt Prow - periodic jobs](https://prow.ci.kubevirt.io?type=periodic&state=failure)

[KubeVirt Prow]: https://prow.ci.kubevirt.io

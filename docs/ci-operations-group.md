# KubeVirt CI Operations Group


## Goal of this document

The current group consists only of a couple of people who mostly are not only handling CI but need to take other tasks as well. This document is intended to make clear what the primary responsibilities are in order to avoid overloading the members of this group with tasks.


## Primary members

<table>
  <tr>
   <td>
   </td>
   <td>
   </td>
   <td>
   </td>
   <td colspan="5" ><b>Responsibilities / Area of interest</b>
   </td>
  </tr>
  <tr>
   <td><b>Human</b>
   </td>
   <td><b>Availability</b>
   </td>
   <td><b>TZ</b>
   </td>
   <td><b>BM</b>
   </td>
   <td><b>Operations</b>
   </td>
   <td><b>Prow</b>
   </td>
   <td><b>kubevirtci</b>
   </td>
   <td><b>SIG Support</b>
   </td>
  </tr>
  <tr>
   <td><a href="https://github.com/brianmcarey">Brian Carey</a>
   </td>
   <td>Mon-Fri
   </td>
   <td>GMT+1
   </td>
   <td>Pri
   </td>
   <td>Pri
   </td>
   <td>Pri
   </td>
   <td>Backup
   </td>
   <td>Backup
   </td>
  </tr>
  <tr>
   <td><a href="https://github.com/dhiller">Daniel Hiller</a>
   </td>
   <td>Mon-Fri
   </td>
   <td>GMT+2
   </td>
   <td>Backup
   </td>
   <td>Sec
   </td>
   <td>Pri
   </td>
   <td>Backup
   </td>
   <td>Pri
   </td>
  </tr>
  <tr>
   <td><a href="https://github.com/enp0s3">Igor Bezukh</a>
   </td>
   <td>Sun-Thu
   </td>
   <td>GMT+3
   </td>
   <td>
   </td>
   <td>Backup
   </td>
   <td>Sec
   </td>
   <td>Backup
   </td>
   <td>Sec
   </td>
  </tr>
  <tr>
   <td><a href="https://github.com/xpivarc">Luboslav Pivarc</a>
   </td>
   <td>Mon-Fri
   </td>
   <td>GMT+2
   </td>
   <td>
   </td>
   <td>Backup
   </td>
   <td>Sec
   </td>
   <td>Backup
   </td>
   <td>Sec
   </td>
  </tr>
</table>


**Note: kubevirtci should be driven by those who use it: Developers. The SIGs.**

### Contact channels

* [#kubevirt-dev (Kubernetes Slack)](https://kubernetes.slack.com/archives/C0163DT0R8X)
* [~~#kubevirt-ci (CoreOS Slack)~~](https://coreos.slack.com/archives/C01EX3K1FGE) - Note: ⚠️ DEPRECATED as it's RH internal

### Jira Boards

* [Task Overview](https://issues.redhat.com/secure/RapidBoard.jspa?rapidView=15140)
* [Sprint board](https://issues.redhat.com/secure/RapidBoard.jspa?rapidView=15438&view=detail)


### Responsibilities of the group

The group is **primarily** responsible for keeping the [KubeVirt CI infrastructure](https://github.com/kubevirt/project-infra/blob/main/docs/infrastructure-components.md#infrastructure-components) operational, such that CI jobs are executed in a timely manner and PRs of [any of the onboarded projects](https://github.com/kubevirt/project-infra/tree/main/github/ci/prow-deploy/files/jobs) are not blocked.

Additional **secondary** responsibilities are:


* keeping an eye on prow job failures and notify members of the sig teams if required
* supporting and educating sig members in CI matters related to prow job configuration
* regularly updating the prow deployment (effectively meaning looking at the [automated bump jobs](https://github.com/kubevirt/project-infra/pulls/kubevirt-bot))
* maintaining cluster nodes (in coordination with and the KNI infrastructure team)
* maintaining cluster configuration (i.e. prow concertation, onboarding and updating other cluster configs inside the [secrets repository](https://github.com/kubevirt/secrets/), also adding secrets that folks can use in their jobs or actions)


### Non-responsibilities

The group is NOT responsible for



* fixing flaky tests, as long as those tests are not caused by the CI infrastructure itself
* fixing the overload of the CI infrastructure if it is caused by misuse of the infrastructure
* improving the runtime of specific lanes as long as the creators of the lane are capable of handling this themselves
* anything else that people outside this group are capable of handling on their own


### Work Classification Labels

**As a** KubeVirt CI member  
**I want to** label work items with the work classification label  
**in order to** prioritize items only the group can handle

| Label | Work Area | What group of people can handle a task of this type |
| --- | --- | --- |
| bare-metal  | Any issue that affects a node  | only KubeVirt CI operations  |
| operations | Any issue that affects the KubeVirt CI clusters | only KubeVirt CI operations |
| prow | Any issue that is related to prow configuration | depends on the scope, if related to prow jobs, anyone can do this |
| kubevirtci | Any issue that is supporting a feature or a bug regarding to kubevirt/kubevirtci | anyone that requires the issue to be addressed |
| sig-support | Any issue that is supporting a project inside kubevirt | anyone, especially someone from the sig that is affected |



## Processes and meetings


### KubeVirt CI Taskforce Sync

The group and a bigger audience interested in KubeVirt CI performs a sync every week on Monday 10AM CEST: 
* [Google Meet](https://meet.google.com/pcy-dnin-ojj)
* [Meeting Notes](https://docs.google.com/document/d/17eKwt7zaPsEcFrP6hVEz2Bvj_DVg3zJLIUMO0XM7kp4/edit?usp=drive_web)


### Backlog grooming

The group performs backlog grooming sessions every two weeks on Monday 11:30 AM - 11:55 AM CEST:
* [Google Meet](https://meet.google.com/orz-vyeh-kob)
* [Meeting Notes](https://docs.google.com/document/d/16N4O73aHzsSLsbaAfqabP6a1-kZave2DlJqoPDofL1c/edit?usp=meetingnotes&showmeetingnotespromo=true)

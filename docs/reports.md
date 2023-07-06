# Reports

Collection of links to search portals, statistics, and reports that are periodically updated over various topics.

## kubevirt/kubevirt

| Link                                                                                                                                                                                          | Docs                                                                                                |
|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------|
| [testgrid](https://testgrid.k8s.io/kubevirt)                                                                                                                                                  | TestGrid holds stats for our periodic and presubmit e2e jobs                                        |
| [ci-health](https://kubevirt.io/ci-health/#kubevirtkubevirt)                                                                                                                                  | Overall health of KubeVirt CI, stats for merge queue, days to merge etc.                            |
| [merge queue (GitHub query)](https://github.com/kubevirt/kubevirt/pulls?q=is%3Apr+is%3Aopen+label%3Aapproved+label%3Algtm+-label%3Ado-not-merge/hold)                                         | GitHub query showing PRs that are actually fit for merge, i.e. they have the required set of labels |
| [flakefinder reports](https://storage.googleapis.com/kubevirt-prow/reports/flakefinder/kubevirt/kubevirt/index.html)                                                                          | Report over test failures that have occurred on merged PRs                                          |
| [flake stats](https://storage.googleapis.com/kubevirt-prow/reports/flakefinder/kubevirt/kubevirt/flake-stats-14days.html) ([PR pending](https://github.com/kubevirt/project-infra/pull/2833)) | stats to show which flakes have hurt the most, and where                                            |
| [presubmits](https://storage.googleapis.com/kubevirt-prow/reports/e2ejobs/kubevirt/kubevirt/presubmits.html)                                                                                  | list of e2e presubmit jobs, their ENV var settings, and whether they gate the merge or not          |
| [periodics](https://storage.googleapis.com/kubevirt-prow/reports/e2ejobs/kubevirt/kubevirt/periodics.html)                                                                                    | list of e2e periodic jobs, their ENV var settings, and whether they gate the merge or not           |
| [quarantined tests](https://storage.googleapis.com/kubevirt-prow/reports/quarantined-tests/kubevirt/kubevirt/index.html)                                                                      | list of quarantined tests, i.e. tests that have the QUARANTINED label                               |

See also [org debugging resources](./debugging-resources/org-debugging-resources.md)

## kubevirt org

| Link                                                                                                                  | Docs                                                     |
|-----------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------|
| [ci search](https://search.ci.kubevirt.io/)                                                                           | CI search makes results of prow jobs searchable          |
| [kubevirt flakefinder reports](https://storage.googleapis.com/kubevirt-prow/reports/flakefinder/index.html) | Entry page for all projects who have flakefinder reports |

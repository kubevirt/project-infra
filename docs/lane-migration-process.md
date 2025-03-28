# Process to migrate a test lane

We regularly migrate (aka bump) lanes to a more recent k8s version, i.e. the [`sig-compute-migrations`] lane is usually only there for the latest k8s version, and whenever there's a new stable provider available, we bump the lane.

Unfortunately this seemingly simple change has a ripple effect caused by the [`statusreconciler`] that takes care about re-triggering jobs in case of configuration changes. In case a required job is changed [`statusreconciler`] triggers for a "new" job that is detected, one instance for every PR. This will cause hundreds on new jobs being created in an instant (depending on the number of open PRs).

Therefore whenever we do such a lane change, we scale down the [`statusreconciler`] to avoid above effect.

# Steps

* whenever a PR that bumps a lane is good to merge, `/hold` the PR at the same time when it gets `/approve`

      /approve

      /hold to wait for the right time to go in
> [!NOTE]
> PRs with a `hold` label will not be merged by [`tide`]
* when the time is right for the bump PR to go in, scale down the deployment

      kubectl scale deployment statusreconciler --replicas=1 -n kubevirt-prow
* unhold the PR

      /unhold

      ready to merge the PR
* use migratestatus (borrowed from `kubernetes/test-infra`) to check whether retiring the old status works, i.e.

      podman run -v /your/token/dir:/etc/tokens --rm quay.io/kubevirtci/migratestatus:v20250326-66cd380 --branch-filter main --github-token-path /etc/tokens/oauth --retire 'pull-kubevirt-e2e-k8s-1.31-sig-compute-migrations' --org 'kubevirt' --repo 'kubevirt' --dry-run=true
* use migratestatus (borrowed from `kubernetes/test-infra`) to retire the old status as above but remove the `--dry-run` flag

[`statusreconciler`]: https://docs.prow.k8s.io/docs/components/optional/status-reconciler/
[`tide`]: https://docs.prow.k8s.io/docs/components/core/tide/
[`sig-compute-migrations`]: https://github.com/kubevirt/project-infra/blob/66cd38052e41a0e24689c9edc4d834e12d3d8828/github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-presubmits.yaml#L964

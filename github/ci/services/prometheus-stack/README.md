# Prometheus stack

Customization and deployment of a prometheus-based monitoring stack, including
[prometheus operator], [alertmanager], [grafana] and [loki]. It uses internally
these bazel [gitops rules].

---
**Note**

Our manifests are based on the [helm chart for kube-prometheus-stack](https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack).

## Layout

We have a setup in which the main components are running in the same cluster as
Prow control plane. Metrics from the rest of clusters are aggregated and
accessible from the main cluster. The architecture of the stack looks like this:

![monitoring stack](monitoring-stack.svg)

These are the components involved:
* Control plane cluster:
  * Grafana: main entry point from the outside to access dashboards and explore
metrics.
  * Thanos query-frontend: retrieves metrics from Thanos querier and makes them
  available to Grafana. It currently uses an in-memory cache to improve the query
  performance.
  * Thanos querier: aggregates metrics from the different Prometheus sidecars and
  from the long-term store.
  * Prometheus and Thanos sidecar: Prometheus is only concerned with scrapping
  metrics and writing blocks to its configured local storage. The sidecar takes
  care of persisting these blocks to the permanent storage and respond to requests
  from querier.
  * Compactor: optimizes indices of blocks uploaded to permanent storage, executes
  downsampling to improve query performance on medium/large time ranges and enforces
  data retention (currently set to 40 days).
  * Alertmanager: receives alerts from Prometheus and sends them to the configured
  receivers (currently only slack).
* Workloads clusters: they only run Prometheus, sidecar and Alertmanager components
described above, the sidecar service must be accessible from the control plane cluster.
We deploy separated instances of Alertmanager instead of leveraging Thanos Ruler to
prevent single points of failure and decoupling the alerting pipelines of each cluster.
* GCS bucket: permanent store of persisted blocks.

## Deployment

You need:
* a kubernetes configuration at `~/.kube/config` with an user allowed to
create [these resources](./manifests).
* [bazelisk] installed.

Then, from the root of project-infra run:
```
$ ./github/ci/services/prometheus-stack/hack/deploy.sh production
```

## Tests

Can be tested locally using [kind] and [bazelisk], from the root of project-infra:
```
$ kind create cluster
$ ./github/ci/services/prometheus-stack/hack/deploy.sh testing
$ bazelisk test //github/ci/services/prometheus-stack/e2e:go_default_test --test_output=all --test_arg=-test.v
```

[gitops rules]: https://github.com/adobe/rules_gitops#:~:text=Bazel%20GitOps%20Rules,kustomize%20overlays%20for%20their%20services.
[prometheus operator]: https://github.com/prometheus-operator/prometheus-operator
[alertmanager]: https://prometheus.io/docs/alerting/latest/alertmanager/
[grafana]: https://grafana.com/
[loki]: https://grafana.com/oss/loki/
[kind]: https://github.com/kubernetes-sigs/kind
[bazelisk]: https://github.com/bazelbuild/bazelisk

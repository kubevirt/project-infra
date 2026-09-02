# Performance Testing

Code changes may affect system behavior, scalability, and performance. So we currently have two types of tests needed to validate kubevirt code. These are:
* Correctness tests - e2e tests verifying expected system behaviors
* Performance tests - e2e tests to test system performance

Correctness tests run as unit and functional tests for each PR, and performance tests run burst density tests on a dedicated cluster in isolation to continually test builds for performance improvements and regressions. Therefore, performance testing constitutes the final layer of protection against regressions.


## Layout

[See Infrastructure layout](infrastructure-components.md#layout)

## Prow cluster
The CI/CD system for performance testing is based on [Prow], a Kubernetes CI system, using the same control plane system as the Correctness tests. Therefore, all test statuses can be viewed in the same testgrid.

## Automated performance cluster tests
Because running performance tests is time consuming, we do not want to run them too often. On the other hand, performing them infrequently implies late identification and accumulation of regressions. Note that they would not run by default on every open PR due to resource, costs, and other constraints. So, we choose the following middle ground:

| Day | |
| ------------- |:-------------:|
| Mon | [kubevirt e2e perf test](../github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-periodics.yaml) @ 00:00, 08:00, 16:00 UTC |
| Tue | kubevirt e2e perf test @ 00:00, 08:00, 16:00 UTC |
| Wed | kubevirt e2e perf test @ 00:00, 08:00, 16:00 UTC |
| Thu | kubevirt e2e perf test @ 00:00, 08:00, 16:00 UTC |
| Fri | kubevirt e2e perf test @ 00:00, 08:00, 16:00 UTC |
| Sat | kubevirt e2e perf test @ 00:00, 08:00, 16:00 UTC |
| Sun | kubevirt e2e perf test @ 00:00, 08:00, 16:00 UTC |

Running a performance jobs every day would:
* help capture regressions daily
* help verify fixes with low latency
* ensure a good release signal

## Test configuration
* KubeVirt e2e perf test is a prow job that runs a medium scale burst density performance test every day creating up to 100 VMIs in the performance cluster.
Waiting for a cool down interval between each scenario. We create up to 100 VMIs per node to avoid overload the node, more info [here](https://2022.fosdem.sojourner.rocks/event/12559).
Testgrid [link](https://testgrid.k8s.io/kubevirt-periodics#periodic-kubevirt-e2e-k8s-1.36-sig-performance).

## Metrics
To analyze the performance regression, we are interested in latency, throughput, and resource utilization metrics. To this end, we continually collect and report on the measurements described above as part of the project testing framework.

### **VM/VMI end-to-end latency**
KubeVirt provides high-level abstractions for users to represent their applications, for example using virtual machine (VM) objects or virtual machine instance (VMI) objects. Where VM are stateful objects that can be stopped and started and VMI represents the basic ephemeral building block of an instance. Note that a VM object can create a VMI object. For the simplicity sake, the CI/CD performance test currently only evaluates the VMI object, but we plan to extend it in the future.

Therefore, the most important metric is the VMI creation latency, which directly impacts the end-user experience.
The VMI creation latency is measured from the time the object is created to the time the VMI object enters in the Running phase. A running VMI means the libvirt domain has been created and the virtual machine OS is booting. So while knowing whether the OS is fully booted or [whether the network is accessible](https://github.com/kubevirt/kubevirt/pull/5946) are also important metrics, they do not depend on the KubeVirt control plane. Hence, these aspects are not being evaluated in CI/CD performance tests at this time.


### **API responsiveness for user-level abstractions**
Another important metric is the tail latency for KubeVirt API operations. The 99th percentile of all API calls should return in less than 1s.

### **VM/VMI creation rate (throughput)**
A high creation rate is crucial for scalability, on a large scale system the system throughput will affect the overall performance. Throughput is measured as the number of VMI running per second.


## Components
* Load generator: [perfscale-load-generator](https://github.com/kubevirt/kubevirt/tree/main/tools/perfscale-load-generator), which is a tool aimed at stressing the Kubernetes and KubeVirt control plane by creating several objects (e.g., VM, VMI, and VMIReplicaSet).

* Monitoring stack: prometheus, node-exporter, and grafana.
   * Prometheus retains data for up to 3 months.
   * Various dashboards including a KubeVirt control-plane dashboard.


## Exposed services

* deck: "periodic-kubevirt-e2e-k8s-\*-sig-performance" job in Prow UI, available at https://prow.ci.kubevirt.io

* grafana: available at https://grafana.ci.kubevirt.io

* ci-search: available at https://search.ci.kubevirt.io


## Contact

Please join our community and help us build the future of KubeVirt! There are many ways to participate. If you’re particularly interested in performance and scalability, you’ll be interested in:

* Joining the scalability “Special Interest Group”, which meets every Thursday at 7 AM Pacific Time at [Zoom meeting](https://zoom.us/j/96406344036). Calendar information [here](https://calendar.google.com/calendar/u/0/embed?src=kubevirt@cncf.io&ctz=GMT).
* Chat with us on Slack via [#virtualization](https://kubernetes.slack.com/?redir=%2Farchives%2FC8ED7RKFE) or [#kubevirt-dev](https://kubernetes.slack.com/archives/C0163DT0R8X) @ kubernetes.slack.com
* Discuss with us on the [kubevirt-dev Google Group](https://groups.google.com/forum/#!forum/kubevirt-dev)

[Prow]: https://github.com/kubernetes-sigs/prow#readme

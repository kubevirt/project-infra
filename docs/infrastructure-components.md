# Infrastructure components

We describe here the elements that form the infrastructure required for projects
under KubeVirt's umbrella. This mainly involves components required to run our
CI/CD systems and related services, like the monitoring stack or applications for
indexing and querying build logs.

The CI/CD system is based on [Prow], kubernetes CI system. For a brief overview
of how Prow runs jobs take a look at ["Life of a Prow Job"]. To see common Prow
usage and interactions flow, see the pull request interactions [sequence diagram].

## Layout

```mermaid
flowchart LR

    subgraph ControlPlaneCluster["kubevirt-prow-control-plane"]

        subgraph KubeVirtProwNS["kubevirt-prow"]
            subgraph ProwCP["Prow Control Plane"]
                    deck
                    hook
                    tide
                    crier
                    OthersCP[...]
            end
            subgraph ProwPlugins["External Plugins"]
                rehearse
                test-subset
                phased
                OtherPlugins[...]
            end
            subgraph ProwSecondary["Secondary Components"]
                gcsweb
                ghproxy
                label-sync
                OtherSecondary[...]
            end
            subgraph BazelCacheCP["Bazel Cache"]
              greenhouse
            end
            DockerProxyCP["Docker Proxy"]
            CertManager["cert-manager"]
        end

        subgraph MonitoringNS["monitoring"]
            Grafana["Grafana"]
        end

        subgraph CISearchNS["ci-search"]
            CISearch["CI Search"]
        end

        subgraph CPJobsNS["kubevirt-prow-jobs"]
            CPJobs["`CI Jobs
            (non-e2e)`"]
        end

        subgraph CPWorkers["Worker Nodes"]
            VMs["9 VMs"]
        end

        ProwCP --> CPJobs
        CPJobs --> BazelCacheCP
        CPJobs --> DockerProxyCP
        CPJobs --> VMs
        CertManager -.-> Grafana
        CertManager -.-> CISearch
        CertManager -.-> ProwCP

    end

    subgraph WorkloadsCluster["prow-workloads"]

        subgraph WCJobsNS["Namespace: kubevirt-prow-jobs"]
            WCJobs["`CI Jobs
                    (e2e tests)`"]
            WCBazel["`Bazel Cache
                    (greenhouse)`"]
            WCDockerProxy["Docker Proxy"]
        end

        subgraph WCMonitoring["Monitoring"]
            WCPrometheus["`Prometheus + Thanos
                            + Node Exporter`"]
        end

        subgraph WCWorkers["Worker Nodes"]
            BMs["11 Bare Metal Servers"]
        end

        WCJobs --> WCBazel
        WCJobs --> WCDockerProxy
        WCJobs --> WCWorkers
        WCJobs --> WCPrometheus

    end

    subgraph ARMCluster["prow-arm64-workloads"]
        subgraph ARMJobsNS["kubevirt-prow-jobs"]
            ARMJobs["ARM Test Jobs"]
        end
        ARMWorkers["Worker Nodes"]
        ARMJobs --> ARMWorkers
    end

    subgraph S390XCluster["prow-s390x-workloads"]
        subgraph S390XJobsNS["kubevirt-prow-jobs"]
            S390XJobs["s390x Test Jobs"]
        end
        S390XWorkers["Worker Nodes"]
        S390XJobs --> S390XWorkers
    end

    subgraph PerfCluster["prow-performance"]
        subgraph PCJobsNS["kubevirt-prow-jobs"]
            PCJobs["`Performance
                    Test Jobs`"]
        end
        subgraph PCMonitoring["Monitoring"]
            PCPrometheus["Prometheus Stack"]
        end
        subgraph PCWorkers["Worker Nodes"]
            PCBMs["6 Bare Metal Servers"]
        end

        PCJobs --> PCPrometheus
        PCJobs --> PCWorkers
    end

    subgraph AMDCluster["amd-workloads"]
        subgraph AMDJobsNS["kubevirt-prow-jobs"]
            AMDJobs["`AMD Test Jobs`"]
        end
    end

    subgraph ExternalServices["External Services"]
        GitHub["GitHub"]
        Quay["`Quay
               (Container Registry)`"]
        GCS[("`GCS
               (Google Cloud Storage)`")]
        Slack["Slack"]
    end

    subgraph ExternalClients["External Clients"]
        CISearchClients["`ci-search clients
                          (search.ci.kubevirt.io)`"]
        DeckClients["`deck clients
                      (prow.ci.kubevirt.io)`"]
        GrafanaClients["`grafana clients
                         (grafana.ci.kubevirt.io)`"]
    end

    %% GitHub webhooks to Prow
    GitHub -->|webhooks| ProwCP

    %% Prow schedules jobs on all clusters
    ProwCP -->|schedules| CPJobs
    ProwCP -->|schedules| WCJobs
    ProwCP -->|schedules| ARMJobs
    ProwCP -->|schedules| S390XJobs
    ProwCP -->|schedules| PCJobs
    ProwCP -->|schedules| AMDJobs

    %% Jobs interact with external services
    CPJobs --> GCS
    WCJobs --> GCS
    WCJobs --> Quay
    ARMJobs --> Quay
    S390XJobs --> Quay
    PCJobs --> Quay

    %% Observability tools
    CISearch <--> GCS
    Grafana --> WCPrometheus
    WCPrometheus --> GCS

    %% External client access
    CISearch <--> CISearchClients
    ProwCP <--> DeckClients
    Grafana <--> GrafanaClients

    %% Notifications
    ProwCP --> Slack

    classDef externalStyle fill:#e1f5ff,stroke:#333,stroke-width:2px
    classDef clusterStyle fill:#fff4e6,stroke:#333,stroke-width:2px
    classDef namespaceStyle fill:#f9f9f9,stroke:#666,stroke-width:1px,stroke-dasharray: 5 5

    class GitHub,Quay,GCS,Slack,CISearchClients,DeckClients,GrafanaClients externalStyle
    class WorkloadsCluster,ARMCluster,S390XCluster,PerfCluster,ControlPlaneCluster,AMDCluster clusterStyle
    class KubeVirtProwNS,MonitoringNS,CISearchNS,CPJobsNS,WCJobsNS,ARMJobsNS,S390XJobsNS,PCJobsNS,AMDJobsNS namespaceStyle
```

**Note:** The [AMD SEV Cluster](amd-sev-cluster.md) (`amd-workloads`) is also part of the infrastructure. See the dedicated documentation for details.

## Prow clusters

Our infrastructure includes several clusters connected directly to Prow, in other
words, Prow can schedule jobs on them. they will be described next, For each of
them this document provides the following fields:

* Prow context: context name of the cluster in the master kubeconfig used by Prow
to access it. It is also the value of the `cluster` field in the Prow jobs that
run on the cluster. The main kubeconfig is part of the [automation secrets], read
more about the [build clusters configuration here].
* Components: which workloads run in the cluster, according to the layout picture
above.
* External services: functionality not running in our infra that is required for
the normal operation of the workloads running in the cluster.
* Connected clusters: for each cluster, which others are required for its normal
operation.
* Exposed services: functionality accessible from outside the cluster, usually as
web applications.

### Control plane

The control plane cluster is a managed cluster on IBM cloud, we only need to care
about the worker nodes. It runs the main components of the infrastructure and
several non-e2e Prow jobs.

#### Context
kubevirt-prow-control-plane

#### Components

* Prow control plane: all the Prow components, including the main microservices
(crier, deck, hook, horologium, prow-controller-manager, tide, sink), several
external plugins (bot-review, rehearse, release-blocker, phased, test-subset) and secondary components
(cherrypicker, gcsweb, ghproxy, label-sync, needs-rebase, prow-exporter,
pushgateway, statusreconciler).

* Grafana: Dashboards for cluster and prow job monitoring 

* Bazel cache ([greenhouse]) speeds up the builds that use bazel.

* [ci-search]: allows us to query CI build logs.

* [docker proxy]: acts as a cache for docker images, reducing the need to access
external registries.

* [cert-manager]: manages TLS certificates for our web services (grafana, ci-search
and deck).

#### External services

* GitHub: Prow receive messages from GH webhooks about events like comments or PR
created. It also sends information about jobs' status and actions to issue comments
or merge PRs.

* GCS: we use buckets to store:
  * Prow build results
  * Build artifacts
  * Thanos blocks

* Quay: we mainly use quay as our container image registry, the images used for
testing and the image artifacts are stored here.

* slack: some notifications are sent to a specific slack channel.

#### Connected clusters

* Workloads, the control plane schedules CI jobs on it.

#### Exposed services

* deck: Prow UI, available at https://prow.ci.kubevirt.io

* grafana: available at https://grafana.ci.kubevirt.io

* ci-search: available at https://search.ci.kubevirt.io

### Workloads

This is a self managed cluster with bare metals as workers. It runs the e2e jobs.

#### Context

prow-workloads

#### Components

* Monitoring stack: prometheus with thanos sidecar and node-exporter

* Bazel cache ([greenhouse]) speeds up the builds that use bazel.

* [docker proxy]: acts as a cache for docker images, reducing the need to access
external registries.

#### External services

* GCS: we use buckets to store:
  * Build artifacts
  * Thanos blocks

* Quay: we mainly use quay as our container image registry, the images used for
testing and the image artifacts are stored here.

#### Connected clusters

Control plane, it sends prow jobs here to be executed and retrieves its state.

#### Exposed services

None.

## External KubeVirtCI clusters

There are several clusters that are also used to run jobs, but in this case
using the external provider feature of KubeVirtCI. The jobs are scheduled in one
of the Prow Clusters described above and during the execution they connect to an
external cluster and run tests on it. There are several considerations to take
into account:
* The jobs won't create an independent KubeVirtCI cluster, so for each cluster
only one job can be run concurrently.
* The credentials to access the cluster in the form of a kubeconfig filemust be
provided separately, they are not included in Prow's main kubeconfig.

### ARM cluster

Used to run ARM test jobs.

### Performance cluster

Runs [performance-related jobs](performance-cluster.md).

[Prow]: https://github.com/kubernetes-sigs/prow#readme
["Life of a Prow Job"]: https://docs.prow.k8s.io/docs/life-of-a-prow-job/
[sequence diagram]: https://docs.prow.k8s.io/images/pr-interactions-sequence.svg
[ci-search]: https://github.com/openshift/ci-search
[docker proxy]: https://github.com/rpardini/docker-registry-proxy
[cert-manager]: https://cert-manager.io/docs/
[greenhouse]: https://github.com/kubernetes/test-infra/tree/1b4b11a/greenhouse
[automation secrets]: https://github.com/kubevirt/secrets/blob/master/secrets.tar.asc
[build clusters configuration here]: ./how-to-add-a-prow-cluster.md

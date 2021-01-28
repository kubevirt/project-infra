# SR-IOV passthrough CNI

This is a CNI ([container network interface](https://github.com/containernetworking/cni))
plugin that passes all physical and virtual functions of a SR-IOV nic into the
namespace of a pod that requests it.

## Where it used

We use this CNI plugin in our CI to test SR-IOV functionality in KubeVirt.

## Notes

We use [multus-cni](https://github.com/intel/multus-cni) to attach the VFs and
PFs to the pod as a secondary network. We then use a `DaemonSet` to deploy the
plugin on all the relevant K8s nodes.

## Directory structure

```
.
├── deploy  # The manifests used to deploy the plugin and its pre-requisites.
│   └── multus-cni.yaml
├── Dockerfile      # The Dockerfile that we use to pack the plugin in.
├── install-plugin  # The script that actually installs the plugin.
├── plugin  # The plugin itself.
│   ├── 90-sriov-passthrough-cni.conf
│   └── sriov-passthrough-cni
└── README.md
```

## Prow job usage:

In order to use the CNI and allocate one SR-IOV PF:
1. Add one of the following annotation to the prow job yaml (both are doing the same):

    [1] `k8s.v1.cni.cncf.io/networks: multus-cni-ns/sriov-passthrough-cni`

    [2] `k8s.v1.cni.cncf.io/networks: '[{"interface":"net1","name":"sriov-passthrough-cni","namespace":"multus-cni-ns"}]'`

2. Check if `pull-kubevirt-e2e-kind-1.17-sriov` job at [3] has requests/limits of `prow/sriov`, if it does, make sure it request only 1 `prow/sriov`. see [4] for example.
This will let k8s know that only one PF is allocated (each PF is represented by one `prow/sriov` resource).
K8s uses these resources in order to know how many jobs can run simultaneously.
Each SR-IOV node of the CI cluster has `prow/sriov` capacity according to the amount of it's available physical PFs.

[3] https://github.com/kubevirt/project-infra/blob/master/github/ci/prow/files/jobs/kubevirt/kubevirt/kubevirt-presubmits.yaml

[4] 
```bash
requests:
  prow/sriov: "1"
limits:
  prow/sriov: "1"
```

* In case 2 PFs are required, please do the following changes:

1. Use the following annotation instead those in the previous [1] section.

    `k8s.v1.cni.cncf.io/networks: '[{"interface":"net1","name":"sriov-passthrough-cni","namespace":"multus-cni-ns"}, {"interface":"net2","name":"sriov-passthrough-cni","namespace":"multus-cni-ns"}]'`

2. Follow [2] of the previous section, but instead of 1 `prow/sriov`, use 2, in order to let k8s know that 2 PFs are
allocated per job, each PF is represented by a `prow/sriov` resource.

```bash
requests:
  prow/sriov: "2"
limits:
  prow/sriov: "2"
```

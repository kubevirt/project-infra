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

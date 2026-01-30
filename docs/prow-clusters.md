# Prow Clusters

[KubeVirt Prow] instance runs CI jobs for most repositories in the GitHub organization.

To support this effort, KubeVirt CI maintains several Prow clusters capable of running the required tests.

[KubeVirt Prow]: https://prow.ci.kubevirt.io/

## KubeVirt Prow Control Plane

This cluster hosted on [IBM Cloud] and leveraging OpenShift is provided by [Red Hat].

In Prow’s configuration, it is referenced using the name `kubevirt-prow-control-plane`.

It is used to host the [Prow] control plane and a couple of additional services.

[Red Hat]: https://www.redhat.com/
[Prow]: https://docs.prow.k8s.io/
[IBM Cloud]: https://cloud.ibm.com/docs/openshift

## Prow Workloads

This cluster hosted on [IBM Cloud Classic] and leveraging bare-metal servers is provided by [Red Hat].

In Prow’s configuration, it is referenced using the name `prow-workloads`.

It is used to run test lanes that are resource-intensive or require specific capabilities such as GPU or [SR-IOV].

[IBM Cloud Classic]: https://cloud.ibm.com/docs/bare-metal
[SR-IOV]: https://docs.kernel.org/PCI/pci-iov-howto.html

## Prow AMD Workloads

This cluster leveraging an [`AMD EPYC 7413`] processor is provided by [AMD].

In Prow’s configuration, it is referenced using the name `amd-workloads`.

It is used to test the support of the [AMD SEV] for confidential VMs in KubeVirt.

[AMD]: https://www.amd.com/
[`AMD EPYC 7413`]: https://www.amd.com/en/products/processors/server/epyc/7003-series/amd-epyc-7413.html
[AMD SEV]: https://www.amd.com/en/developer/sev.html

## Prow ARM64 Workloads

This cluster hosted on [AWS] and leveraging an [`AWS Graviton2`] processor is provided by [ARM].

In Prow’s configuration, it is referenced using the name `prow-arm64-workloads`.

It is used to test the support of the [ARM64] architecture in KubeVirt.

[ARM]: https://www.arm.com/
[AWS]: https://aws.amazon.com/
[`AWS Graviton2`]: https://aws.amazon.com/ec2/graviton/
[ARM64]: https://docs.kernel.org/arch/arm64/index.html

## Prow HyperV Workloads

This cluster hosted on [Azure] and leveraging Microsoft Azure Linux is provided by [Microsoft].

In Prow’s configuration, it is referenced using the name `prow-hyperv-workloads`.

It is used to test the support of the [HyperV] hypervisor in KubeVirt.

[Microsoft]: https://www.microsoft.com/
[Azure]: https://azure.microsoft.com/
[HyperV]: https://learn.microsoft.com/windows-server/virtualization/hyper-v/

## Prow s390x Workloads

This cluster leveraging an `IBM/S390` processor is provided by [IBM].

In Prow’s configuration, it is referenced using the name `prow-s390x-workloads`.

It is used to test the support of the [s390x] architecture in KubeVirt.

[IBM]: https://www.ibm.com/
[s390x]: https://docs.kernel.org/arch/s390/index.html

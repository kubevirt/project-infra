vm-image-builder image
===============

KubeVirt CI general purpose image for building Customized ephemeral container-disk images for KubeVirt VM's.

To be more specific, for building VM images in kubevirtci/cluster-provision/images/vm-image-builder.

How to setup the image
-----------------
The container is required to run in privileged mode, as /dev/kvm is needed. The shell script `start_libvirt.sh` would start the libvirtd related daemon, then libvirt related cli can works.

```yaml
spec:
    containers:
    - command:
      -  /usr/local/bin/start_libvirtd.sh && /usr/local/bin/runner.sh
      ...
      securityContext:
        privileged: true

```

FROM quay.io/kubevirtci/bootstrap:v20221222-8f66e7e

RUN dnf install -y \
    cloud-utils \
    libguestfs \
    libguestfs-tools-c \
    libvirt \
    qemu-img \
    virt-install && \
    dnf clean all && \
    rm -rf /var/cache /var/log/dnf* /var/log/yum.*

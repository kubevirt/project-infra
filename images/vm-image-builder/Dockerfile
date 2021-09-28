FROM quay.io/kubevirtci/bootstrap:v20210924-4c47964

RUN dnf install -y \
    cloud-utils \
    libguestfs \
    libguestfs-tools-c \
    libvirt \
    qemu-img \
    virt-install && \
    dnf clean all && \
    rm -rf /var/cache /var/log/dnf* /var/log/yum.*

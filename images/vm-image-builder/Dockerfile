FROM quay.io/kubevirtci/bootstrap:v20230502-bfaa042

RUN dnf install -y \
    cloud-utils \
    libguestfs \
    libguestfs-tools-c \
    libvirt \
    qemu-img \
    genisoimage \
    virt-install && \
    dnf clean all && \
    rm -rf /var/cache /var/log/dnf* /var/log/yum.*

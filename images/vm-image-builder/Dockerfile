FROM quay.io/kubevirtci/bootstrap:v20240607-febc467

RUN dnf install -y \
    cloud-utils \
    libguestfs \
    libguestfs-tools-c \
    libvirt \
    qemu-img \
    qemu-system-aarch64 \
    qemu-system-s390x \
    genisoimage \
    virt-install && \
    dnf clean all && \
    rm -rf /var/cache /var/log/dnf* /var/log/yum.*

COPY qemu.conf /etc/libvirt/qemu.conf
COPY start_libvirtd.sh /usr/local/bin/start_libvirtd.sh

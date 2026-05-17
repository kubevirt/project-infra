#!/usr/bin/env bash
set -euo pipefail

# Set qemu.conf key to a value, whether commented, uncommented, or missing.
set_qemu_conf_value() {
    local key="${1:?}"
    local value="${2:?}"
    local file="/etc/libvirt/qemu.conf"
    local pattern="^[[:space:]]*#?[[:space:]]*${key}[[:space:]]*="

    if grep -Eq "${pattern}" "${file}"; then
        sed -Ei "s|${pattern}.*|${key} = ${value}|" "${file}"
    else
        printf "%s = %s\n" "${key}" "${value}" >> "${file}"
    fi
}

mkdir -p /run/libvirt /var/lib/libvirt/dnsmasq

# Required container-specific libvirt settings.
set_qemu_conf_value "user" '"root"'
set_qemu_conf_value "group" '"root"'
set_qemu_conf_value "security_driver" '"none"'
set_qemu_conf_value "dynamic_ownership" "0"
set_qemu_conf_value "remember_owner" "0"

# Start required daemons in the container.
for daemon in virtlogd virtlockd virtnetworkd virtstoraged virtqemud; do
    if ! pgrep -x "${daemon}" >/dev/null 2>&1; then
        "${daemon}" -d
    fi
done

export LIBVIRT_DEFAULT_URI=qemu:///system
virsh net-start default >/dev/null 2>&1 || true
virsh net-autostart default >/dev/null 2>&1 || true

echo "Libvirt bootstrap completed."
echo "Use 'export LIBVIRT_DEFAULT_URI=qemu:///system' in your shell if needed."

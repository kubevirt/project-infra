#!/bin/bash

set -euo pipefail

MIRROR_URL='https://storage.googleapis.com/kubevirtci-crio-mirror'

function usage {
    cat <<EOF
usage: $0 <kubevirtci k8s provider directory>

    Keep CRI-O of each kubevirtci provider in sync with the CRI-O mirror
    (see hack/mirror-crio.sh):
    - align the CRI-O version with the Kubernetes version of the provider,
      as soon as that CRI-O version is mirrored
    - switch the CRI-O channel of the provider from prerelease to stable,
      as soon as its CRI-O version is mirrored

EOF
}

function is_mirrored {
    curl --silent --fail --head --output /dev/null "${MIRROR_URL}/isv_cri-o_stable_v$1/repodata/repomd.xml"
}

if [ "$#" -ne 1 ] || [ ! -d "$1" ]; then
    usage
    exit 1
fi

for provision_script in "$1"/*/k8s_provision.sh; do
    provider_version=$(basename "$(dirname "${provision_script}")")
    # matches both CRIO_VERSION=1.36 (optionally exported) and CRIO_VERSION='1.37'
    crio_version=$(sed -nE "s/^(export )?CRIO_VERSION=['\"]?([0-9]+\.[0-9]+)['\"]?$/\2/p" "${provision_script}")
    if [ -z "${crio_version}" ]; then
        echo "No CRIO_VERSION found in ${provision_script}" >&2
        exit 1
    fi
    # only bump when CRI-O is behind the provider, a newer CRI-O might have been pinned on purpose
    if [ "$(printf '%s\n' "${crio_version}" "${provider_version}" | sort -V | tail -n1)" != "${crio_version}" ]; then
        if is_mirrored "${provider_version}"; then
            echo "Bumping CRI-O from ${crio_version} to ${provider_version} for provider ${provider_version}"
            sed -i -E "s/^((export )?CRIO_VERSION=['\"]?)${crio_version//./\\.}(['\"]?)$/\1${provider_version}\3/" "${provision_script}"
            crio_version=${provider_version}
        else
            echo "CRI-O ${provider_version} is not mirrored yet, keeping CRI-O ${crio_version} for provider ${provider_version}"
        fi
    fi
    # on the stable channel, the provider installs CRI-O from the mirror
    if grep -qE "^CRIO_CHANNEL=['\"]?prerelease['\"]?$" "${provision_script}"; then
        if is_mirrored "${crio_version}"; then
            echo "Switching CRI-O ${crio_version} from prerelease to stable for provider ${provider_version}"
            sed -i -E "s/^(CRIO_CHANNEL=['\"]?)prerelease(['\"]?)$/\1stable\2/" "${provision_script}"
        else
            echo "CRI-O ${crio_version} is not mirrored yet, keeping the prerelease channel for provider ${provider_version}"
        fi
    fi
done

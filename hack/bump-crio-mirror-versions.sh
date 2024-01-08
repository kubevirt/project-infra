#!/bin/bash

PROJECT_INFRA_ROOT=$(readlink --canonicalize $(pwd))
KUBEVIRTCI_PERIODICS="$PROJECT_INFRA_ROOT/github/ci/prow-deploy/files/jobs/kubevirt/kubevirtci/kubevirtci-periodics.yaml"
CURRENT_VERSIONS=$(grep -A 1 "CRIO_VERSIONS" $KUBEVIRTCI_PERIODICS  | grep value | cut -d \" -f 2)
LATEST_VERSION="1.$(curl -s https://api.github.com/repos/cri-o/cri-o/releases/latest | grep tag_name | cut -d : -f 2,3 | cut -d . -f 2)"

if [[ $(echo "$CURRENT_VERSIONS" | grep $LATEST_VERSION) ]]; then
    echo "Mirror includes latest cri-o version"
    exit 0
else
    echo "Mirror does not include latest cri-o version $LATEST_VERSION"
    sed -i "s/${CURRENT_VERSIONS}/${CURRENT_VERSIONS},${LATEST_VERSION}/" $KUBEVIRTCI_PERIODICS
fi

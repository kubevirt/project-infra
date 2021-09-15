#!/usr/bin/env bash

set -euo pipefail
set -x

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
PROJECT_INFRA_ROOT=$(readlink --canonicalize ${BASEDIR}/..)

# won't work as image names differ from directory names
#IMAGES=$(
#    ( cd "$PROJECT_INFRA_ROOT/images" && find . -mindepth 1 -maxdepth 1 -type d -print ) & ( cd "$PROJECT_INFRA_ROOT/robots/cmd" && find . -mindepth 1 -maxdepth 1 -type d -print ) ;
#)
IMAGES=( autoowners bootstrap ci-usage-exporter flakefinder golang indexpagecreator kubekins-e2e kubevirt-infra-bootstrap kubevirt-userguide prow-deploy pr-creator release-blocker release-querier )
for image_dir in "${IMAGES[@]}"; do
    image_name="quay.io/kubevirtci/${image_dir/#\.\//}"
    if ! "$PROJECT_INFRA_ROOT/hack/update-jobs-with-latest-image.sh" "$image_name"; then
        echo "Failed to update prow jobs using image $image_name"
    else
        echo "Updated prow jobs using image $image_name"
    fi
done

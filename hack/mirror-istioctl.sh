#!/bin/bash

# Usage
# ISTIO_VERSIONS=1.9.0,1.10.1 ./mirror-istioctl.sh

set -e

LOCAL_ISTIOCTL_DIR=istioctl-mirror

mkdir -p $LOCAL_ISTIOCTL_DIR

(
    cd $LOCAL_ISTIOCTL_DIR

    # Loop over comma-separated list of istio versions
    for i in $(echo $ISTIO_VERSIONS | sed "s/,/ /g")
    do
        echo "Getting Istio version $i"
        curl -L https://istio.io/downloadIstio | ISTIO_VERSION=$i sh -
    done
)

gcloud auth activate-service-account --key-file=/etc/gcs/service-account.json
gsutil rsync -d -r $LOCAL_ISTIOCTL_DIR gs://$BUCKET_DIR

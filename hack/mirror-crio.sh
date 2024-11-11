#!/bin/bash

# Usage:
# ./mirror-crio.sh


set -e

LOCAL_MIRROR_DIR="crio-mirror"

dnf install dnf-utils -y

mirror_crio_repo_for_version () {
    OLD_CRIO_REPO_VERSIONS="1.22,1.23,1.24,1.25,1.26,1.27"

    if [[ $OLD_CRIO_REPO_VERSIONS == *"$1"* ]]; then
        BASE_REPOID="devel_kubic_libcontainers_stable"
        if [ -z "$1" ]; then
            CRIO_SUBDIR=""
            REPOID=$BASE_REPOID
        else
            CRIO_SUBDIR=":cri-o:$1"
            REPOID="${BASE_REPOID}_cri-o_$1"
        fi
        # cri-o builds > v1.24.1 are broken for Centos_8_Stream
        # cri-o > v1.24.2 is required to avoid https://github.com/cri-o/cri-o/issues/5889
        # Use CentOS_8 build for cri-o v1.24
        if [[ $1 = 1.24 ]]; then
            OS="CentOS_8"
        else
            OS="CentOS_8_Stream"
        fi
        curl -L -o $REPOID.repo https://download.opensuse.org/repositories/devel:kubic:libcontainers:stable$CRIO_SUBDIR/$OS/devel:kubic:libcontainers:stable$CRIO_SUBDIR.repo
        reposync -c $REPOID.repo -p ./$LOCAL_MIRROR_DIR -n --repoid=$REPOID --download-metadata
    else
        echo "Mirroring from new pkgs.k8s.io repos"
        REPOID="isv_kubernetes_addons_cri-o_stable_v$1"
        curl -L -o $REPOID.repo https://download.opensuse.org/repositories/isv:/kubernetes:/addons:/cri-o:/stable:/v$1/rpm/isv:kubernetes:addons:cri-o:stable:v$1.repo
        reposync -c $REPOID.repo -p ./$LOCAL_MIRROR_DIR --repoid=$REPOID --download-metadata
    fi
}


# First mirror the shared stable repository
mirror_crio_repo_for_version

# Loop over comma-separated list of cri-o versions
for i in $(echo $CRIO_VERSIONS | sed "s/,/ /g")
do
    mirror_crio_repo_for_version $i
done

gsutil rsync -d -r $LOCAL_MIRROR_DIR gs://$BUCKET_DIR

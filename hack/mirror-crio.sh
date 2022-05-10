#!/bin/bash

# Usage:
# ./mirror-crio.sh


set -e

dnf install dnf-utils -y

mirror_crio_repo_for_version () {
    if [ -z "$1" ]
    then
        CRIO_SUBDIR=""
        REPOID=$BASE_REPOID
    elif [[ $1 = *":"* ]]
    then    
	CRIO_SUBDIR=":cri-o:$1"
	REPOID_VER=$(echo $1 | sed 's/:/_/g')
	REPOID="${BASE_REPOID}_cri-o_$REPOID_VER"
    else
        CRIO_SUBDIR=":cri-o:$1"
        REPOID="${BASE_REPOID}_cri-o_$1"
    fi
    curl -L -o $REPOID.repo https://download.opensuse.org/repositories/devel:kubic:libcontainers:stable$CRIO_SUBDIR/$OS/devel:kubic:libcontainers:stable$CRIO_SUBDIR.repo
    reposync -c $REPOID.repo -p ./$LOCAL_MIRROR_DIR -n --repoid=$REPOID --download-metadata
}

# First mirror the shared stable repository
mirror_crio_repo_for_version

# Loop over comma-separated list of cri-o versions
for i in $(echo $CRIO_VERSIONS | sed "s/,/ /g")
do
    mirror_crio_repo_for_version $i
done

gsutil rsync -d -r $LOCAL_MIRROR_DIR gs://$BUCKET_DIR

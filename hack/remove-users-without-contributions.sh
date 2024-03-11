#!/usr/bin/env bash

set -exuo pipefail

if [[ "$1" =~ -h ]]; then
    cat << EOF
usage: $0 <input-file>

    Removes all users without contributions (taken from the input file) from the orgs.yaml

    input file in csv format is manually fetched from KubeVirt devstats:
    https://kubevirt.devstats.cncf.io/d/48/users-statistics-by-repository-group?orgId=1&from=now-1y&to=now&var-period=w&var-metric=activity&var-repogroup_name=All&var-users=All

    then from the top panel select "Inspect" > "Data" and select "Download CSV"
EOF
    exit 0
fi

if [ ! -f "$1" ]; then
    echo "file $1 doesn't exist"
    exit 1
fi
input_csv_file="$1"

# generate input file from csv where each user is in a separate line
head -1 "$input_csv_file" \
    | tr ',' "\n" \
    | sed 's/"//g' \
    | sort -u \
    > /tmp/users-with-contributions.txt

### we add some user accounts, so that they never get removed

# bots (kubevirt, openshift, the linux foundation)
cat << EOF >> /tmp/users-with-contributions.txt
kubevirt-bot
kubevirt-commenter-bot
kubevirt-snyk
openshift-ci-robot
openshift-merge-robot
thelinuxfoundation
EOF

# KubeVirt org admins (security measure so that we don't lose GitHub org access)
cat << EOF >> /tmp/users-with-contributions.txt
brianmcarey
davidvossel
dhiller
fabiand
rmohr
EOF

# users with invisible contributions (i.e. OSPO, KubeVirt community manager etc)
cat << EOF >> /tmp/users-with-contributions.txt
aburdenthehand
jberkus
EOF

# iterate over org users (members and admins), grep over contributors list, remove every user not found
(
    for username in $(
        { yq read github/ci/prow-deploy/files/orgs.yaml orgs.kubevirt.members & \
          yq read github/ci/prow-deploy/files/orgs.yaml orgs.kubevirt.admins ; \
        } | sed 's/^- //' | sort -u); do
        echo $username "$(grep -i -c $username /tmp/users-with-contributions.txt)"
    done
) | grep ' 0$' \
  | sed 's/ 0$//' \
  > /tmp/users-without-contributions.txt

# remove all users without contributions
for user in $(cat /tmp/users-without-contributions.txt); do
    sed -i -E '/\s+- '"$user"'/d' github/ci/prow-deploy/kustom/base/configs/current/orgs/orgs.yaml
done

echo "Users without contributions:"
cat /tmp/users-without-contributions.txt

#!/bin/bash
# Source: https://github.com/openshift/release/blob/450e9c1f62f53999809c45c22c46417d1e11c1c9/core-services/ipi-deprovision/gcp.sh
set -o errexit
set -o nounset
set -o pipefail

trap 'CHILDREN=$(jobs -p); if test -n "${CHILDREN}"; then kill ${CHILDREN} && wait; fi' TERM

function queue() {
  local LIVE="$(jobs | wc -l)"
  while [[ "${LIVE}" -ge 10 ]]; do
    sleep 1
    LIVE="$(jobs | wc -l)"
  done
  echo "${@}"
  "${@}" &
}

function deprovision() {
  WORKDIR="${1}"
  timeout 60m openshift-install --dir "${WORKDIR}" --log-level info destroy cluster && touch "${WORKDIR}/success" || touch "${WORKDIR}/failure"
}

logdir="/tmp/deprovision"
mkdir -p "${logdir}"


gce_cluster_age_cutoff="$(TZ=":America/Los_Angeles" date --date="${CLUSTER_TTL}-8 hours" '+%Y-%m-%dT%H:%M%z')"
echo "deprovisioning clusters with a creationTimestamp before ${gce_cluster_age_cutoff} in GCE ..."
export CLOUDSDK_CONFIG=/tmp/gcloudconfig
mkdir -p "${CLOUDSDK_CONFIG}"
gcloud auth activate-service-account --key-file="${GOOGLE_APPLICATION_CREDENTIALS}"

echo "GCP project: ${GCP_PROJECT}"

export FILTER="creationTimestamp.date('%Y-%m-%dT%H:%M%z')<${gce_cluster_age_cutoff} AND autoCreateSubnetworks=false AND name~'ci-'"
for network in $( gcloud --project="${GCP_PROJECT}" compute networks list --filter "${FILTER}" --format "value(name)" ); do
  infraID="${network%"-network"}"
  region="$( gcloud --project="${GCP_PROJECT}" compute networks describe "${network}" --format="value(subnetworks[0])" | grep -Po "(?<=regions/)[^/]+" || true )"
  if [[ -z "${region:-}" ]]; then
    region=us-east1
  fi
  workdir="${logdir}/${infraID}"
  mkdir -p "${workdir}"
  cat <<EOF >"${workdir}/metadata.json"
{
  "infraID":"${infraID}",
  "gcp":{
    "region":"${region}",
    "projectID":"${GCP_PROJECT}"
  }
}
EOF
  echo "will deprovision GCE cluster ${infraID} in region ${region}"
done

clusters=$( find "${logdir}" -mindepth 1 -type d )
for workdir in $(shuf <<< ${clusters}); do
  queue deprovision "${workdir}"
done

FAILED="$(find ${clusters} -name failure -printf '%H\n' | sort)"
if [[ -n "${FAILED}" ]]; then
  echo "Deprovision failed on the following clusters:"
  xargs --max-args 1 basename <<< $FAILED
  exit 1
fi

# Prune GCP CSI driver leftovers
DISK_FILTER="creationTimestamp.date('%Y-%m-%dT%H:%M%z')<${gce_cluster_age_cutoff} AND NOT users:* AND labels.list(show='keys')~'kubernetes-io-cluster-ci-'"
gcloud --project="${GCP_PROJECT}" compute disks list --filter "${DISK_FILTER}" --uri \
  | xargs -r --max-procs='10' gcloud --project="${GCP_PROJECT}" compute disks delete --quiet

SNAPSHOT_FILTER="creationTimestamp.date('%Y-%m-%dT%H:%M%z')<${gce_cluster_age_cutoff} AND description:'Snapshot created by GCE-PD CSI Driver' AND sourceDisk.basename():pvc-"
gcloud --project="${GCP_PROJECT}" compute snapshots list --filter "${SNAPSHOT_FILTER}" --uri \
  | xargs -r --max-procs='10' gcloud --project="${GCP_PROJECT}" compute snapshots delete --quiet

echo "Deprovision finished successfully"

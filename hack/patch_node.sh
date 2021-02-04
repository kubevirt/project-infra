#!/bin/bash -e

# This script can be used in order to create/remove/update k8s extended resources named prow/sriov
# see https://kubernetes.io/docs/tasks/administer-cluster/extended-resource-node for more details.
# The resource will be created node wise, so each sriov node should be configured according
# the desired capacity (according how many PFs it has, and how many jobs are actually desired to run simultaneously).
# After creation, the resource would appear as allocatable on node's yaml after around 20 seconds.
# The following command can be used in order to check it was created successfully.
# timeout 60s bash -c "until oc get node $NODE -oyaml | grep prow/sriov | grep $CAPACITY | wc -l | grep 2; do sleep 1; done"

# Usage:
# ./patch_node.sh <NODE_NAME> <CAPACITY>

PROXY_PORT=9999
if which oc &> /dev/null; then
  BINARY=oc
elif which kubectl &> /dev/null; then
  BINARY=kubectl
else
  echo "error: oc / kubectl not found"
  exit 1
fi

function finish {
  rc=$?
  if jobs -p | xargs kill; then
    echo -e "\nProxy deleted"
  fi
  exit "$rc"
}

function validate_parameters {
  local node=$1
  local capacity=$2
  if ! $BINARY get node "$node" &> /dev/null; then
    echo "error: node $node not found"
    exit 1
  fi

  if [ -z "${capacity##*[!0-9]*}" ]; then
    echo "error: capacity $capacity must be greater or equal 0"
    exit 1
  fi
}

function run_proxy {
  $BINARY proxy -p $PROXY_PORT &
  sleep 3
  jobs 1 | grep -q Running
}

function patch_node {
  local node=$1
  local capacity=$2
  if [ "$capacity" -ne 0 ]; then
    curl --header "Content-Type: application/json-patch+json" \
      --request PATCH \
      --data '[{"op": "add", "path": "/status/capacity/prow~1sriov", "value": "'$capacity'"}]' \
      http://localhost:$PROXY_PORT/api/v1/nodes/"$node"/status
  else
    curl --header "Content-Type: application/json-patch+json" \
      --request PATCH \
      --data '[{"op": "remove", "path": "/status/capacity/prow~1sriov"}]' \
      http://localhost:$PROXY_PORT/api/v1/nodes/"$node"/status
  fi
}

function main() {
  local node=$1
  local capacity=$2
  if [ -z "$node" ] || [ -z "$capacity" ]; then
    echo "syntax error, use: $0 <NODE_NAME> <CAPACITY>"
    exit 1
  fi

  validate_parameters "$node" "$capacity"

  trap finish EXIT
  run_proxy

  patch_node "$node" "$capacity"
}

main "$@"

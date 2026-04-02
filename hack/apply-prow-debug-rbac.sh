#!/usr/bin/env bash

set -x
set -euo pipefail

rbac_manifest="$(dirname "$0")/../github/ci/prow-deploy/kustom/base/manifests/local/prow-debug-rbac.yaml"

clusters=( kubevirt-prow-control-plane prow-workloads )
namespaces=( kubevirt-prow-jobs kubevirt-prow )

for cluster in "${clusters[@]}"; do
    for namespace in "${namespaces[@]}"; do
        kubectl --context "$cluster" -n "$namespace" apply -f "$rbac_manifest"
    done
done

#!/bin/bash

# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright the KubeVirt Authors.

set -eu -o pipefail

base_dir=$(realpath "$(dirname "${BASH_SOURCE[0]}")/..")
manifests_dir=${base_dir}/kustom/base/manifests/upstream
manifests_url=https://github.com/kubernetes-sigs/prow/raw/main/config/prow
manifests_base=starter-gcs.yaml

curl(){ command curl -sSL --fail-with-body --retry 3 "$@"; }

mkdir -p "${manifests_dir}"

cd "${manifests_dir}"

# Cleanup previous manifests
rm -f *.yml *.yaml

# Download upstream deployment manifests
curl -O "${manifests_url}/cluster/starter/${manifests_base}" \
     -O "${manifests_url}/cluster/prowjob-crd/prowjob_customresourcedefinition.yaml"

# Update image tags to the latest version
latest_prow_version=$(
    curl https://us-docker.pkg.dev/v2/k8s-infra-prow/images/prow-controller-manager/tags/list \
        | yq -r '
            .manifest | to_entries
            | map(.value) | sort_by(.timeUploadedMs)
            | .[-1].tag[]
            | select(test("^(v?\d{8}-(?:v\d(?:[.-]\d+)*-g)?[0-9a-f]{6,10})$"))
          '
)

sed -re "s|v[0-9]{8}-[a-f0-9]{6,10}|${latest_prow_version}|" -i "${manifests_base}"

# Split the monolitic manifest into subfiles using the naming patterns:
# - name_kind.yml for resources in the main prow namespace
# - name_kind_namespace.yml for resources in the test pods namespace

upstream_prow_config=$(
    yq -r '
      select(.kind == "ConfigMap" and .metadata.name == "config")
      | .data."config.yaml"
    ' "${manifests_base}"
)
upstream_prow_namespace=$(yq -r '.prowjob_namespace' <<<"${upstream_prow_config}")

export upstream_prow_namespace

yq --prettyPrint --split-exp '
  strenv(upstream_prow_namespace) as $main_namespace
  | (.metadata.name) as $name
  | (.kind | downcase) as $kind
  | (.metadata.namespace // $main_namespace) as $namespace
  | "\($name)_\($kind)_\($namespace)"
  | sub("_\($main_namespace)$", "")
' "${manifests_base}" \

rm -f "${manifests_base}"

cd - >/dev/null

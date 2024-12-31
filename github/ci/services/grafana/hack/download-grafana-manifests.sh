#!/usr/bin/env bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright the KubeVirt Authors.
#
#
# see https://grafana.github.io/grafana-operator/docs/installation/kustomize/

GRAFANA_OPERATOR_VERSION="${GRAFANA_OPERATOR_VERSION:-v5.4.1}"
GRAFANA_KUSTOMIZE_DIR="/tmp/grafana-operator"

mkdir -p "${GRAFANA_KUSTOMIZE_DIR}"
flux pull artifact "oci://ghcr.io/grafana/kustomize/grafana-operator:${GRAFANA_OPERATOR_VERSION}" --output "${GRAFANA_KUSTOMIZE_DIR}"


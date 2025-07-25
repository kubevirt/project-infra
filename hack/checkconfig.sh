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

set -euo pipefail

project_infra_dir="$(realpath "$(dirname "${BASH_SOURCE[0]}")/..")"

podman run --rm \
    -v "${project_infra_dir}:/project-infra" \
    us-docker.pkg.dev/k8s-infra-prow/images/checkconfig:v20250709-d01b8af18 \
    --config-path /project-infra/github/ci/prow-deploy/files/config.yaml \
    --job-config-path /project-infra/github/ci/prow-deploy/files/jobs \
    --plugin-config /project-infra/github/ci/prow-deploy/files/plugins.yaml \
    --strict

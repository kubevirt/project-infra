#!/usr/bin/env bash
# Copyright 2019 The Kubernetes Authors.
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

# Runs sigs.k8s.io/prow/pkg/pj-on-kind.sh with config arguments specific to the prow.k8s.io instance.

set -o errexit
set -o nounset
set -o pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
export CONFIG_PATH="${root}/github/ci/prow-deploy/files/config.yaml"
export JOB_CONFIG_PATH="${root}/github/ci/prow-deploy/files/jobs"

bash <(curl -s https://raw.githubusercontent.com/kubernetes-sigs/prow/main/pkg/pj-on-kind.sh) "$@"

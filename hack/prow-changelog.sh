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
# Copyright The KubeVirt Authors.
#
#
# When executed inside the kubevirt/project-infra repository, this script
# extracts the change in commit ids from bump-prow.sh
# and then lists the matching commits from kubernetes-sigs/prow in oneline
# format, enhancing the commit ids with links into the latter repository.
#
# Requirements:
# * git installed
# * github.com/kubernetes-sigs/prow checked out into ../../kubernetes-sigs/prow

set -e
set -u
set -o pipefail

commit_range=$(
    git diff -- hack/bump-prow.sh | \
        grep -E '^[+-]\s.*v[0-9]+-([a-f0-9]+).*$' |\
        sed -E 's#^[+-]\s.*v[0-9]+-([a-f0-9]+).*$#\1#g'
)
changes=$(
    (
        cd ../../kubernetes-sigs/prow
        git log --format=oneline --abbrev-commit "${commit_range/$'\n'/..}"
    ) | sed -E 's#^([a-f0-9]+)#* \[\1\]\(https://github.com/kubernetes-sigs/prow/commit/\1\)#g' \
      | sed -E 's/ (#[0-9]+) / kubernetes-sigs\/prow\1 /g'
)
cat <<EOF
Bump Prow

/cc @kubevirt/prow-job-taskforce

Changes:
${changes}
EOF
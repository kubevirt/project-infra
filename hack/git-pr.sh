#!/bin/bash
# Copyright 2018 The Kubernetes Authors.
# Copyright 2020 The KubeVirt Authors.
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
# Taken from https://github.com/kubernetes/test-infra/blob/4d7f26e59a5e186eef3a7de55486b7a40bbd79d7/hack/autodeps.sh
# and modified for kubevirt.

set -o nounset
set -o errexit
set -o pipefail

usage() {
    echo "Usage: $(basename "$0") -c \"<command>\" [-s \"<summary>\"] [-l <github-login>] [-t </path/to/github/token>] [-T <target-branch>] [-p </path/to/github/repo>] [-n \"<git-name>\"] [-e <git-email>]  [-b <pr-branch>] [-o <org>] [-r <repo>] [-m </path/where/command/should/be/run>]">&2
}

command=
summary=
user=kubevirt-bot
token=/etc/github/oauth
repo_path=$(pwd)
git_name=kubevirt-bot
git_email=kubevirtbot@redhat.com
branch=autoupdate
org=kubevirt
repo=kubevirt
command_path=$(pwd)
targetbranch=master

while getopts ":c:s:l:t:T:p:n:e:b:o:r:m" opt; do
    case "${opt}" in
        c )
            command="${OPTARG}"
            ;;
        s )
            summary="${OPTARG}"
            ;;
        l )
            user="${OPTARG}"
            ;;
        t )
            token="${OPTARG}"
            ;;
        p )
            repo_path="${OPTARG}"
            ;;
        n )
            git_name="${OPTARG}"
            ;;
        e )
            git_email="${OPTARG}"
            ;;
        b )
            branch="${OPTARG}"
            ;;
        T )
            targetbranch="${OPTARG}"
            ;;
        o )
            org="${OPTARG}"
            ;;
        r )
            repo="${OPTARG}"
            ;;
        m )
            command_path="${OPTARG}"
            ;;
        \? )
            usage
            exit 1
            ;;
    esac
done

if [ -z "${command}" ]; then
    usage
    exit 1
fi

if [ -z "${summary}" ]; then
    summary="Run ${command}"
fi

cd "${command_path}"
eval "${command}"

cd "${repo_path}"
echo "git config user.name=${git_name} user.email=${git_email}..." >&2
git config user.name "${git_name}"
git config user.email "${git_email}"

if ! git config user.name &>/dev/null && git config user.email &>/dev/null; then
    echo "ERROR: git config user.name, user.email unset. No defaults provided" >&2
    exit 1
fi

git add -A
if git diff --name-only --exit-code HEAD; then
    echo "Nothing changed" >&2
    exit 0
fi

git commit -s -m "${summary}"
git push -f "https://${user}@github.com/${user}/${repo}.git" HEAD:"${branch}"

echo "Creating PR to merge ${user}:${branch} into master..." >&2
pr-creator \
    --github-token-path="${token}" \
    --org="${org}" --repo="${repo}" --branch="${targetbranch}" \
    --title="${summary}" --match-title="${summary}" \
    --body="Automatic run of \"$command\". Please review" \
    --source="${user}":"${branch}" \
    --confirm

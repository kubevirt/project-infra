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
set -x

# Constants and default values
body=
branch=autoupdate
command=
command_path=${PWD}
description_command=
dry_run=
git_author=kubevirt-bot
git_email=kubevirtbot@redhat.com
head_branch=
labels=
missing_labels=
org=kubevirt
progname=${0##*/}
release_note_none=
repo=kubevirt
repo_path=${PWD}
summary=
targetbranch=master
title=
token=/etc/github/oauth
user=kubevirt-bot

usage() {
    cat <<EOF
Wrapper script around the \`pr-creator\` tool. It runs a command, commits the
changes and creates a Pull Request to a GitHub repository.

Usage:
  ${progname} <options>

Short options are deprecated.

Options:
  --author, -n "<git-author>"
    The git author name to use when committing changes.
    (Default: ${git_author:-unset})

  --body, -B "<body-text>"
    Custom body text for the PR. If provided, it overrides the body generated
    by the <description-command>.
    (Default: "Automatic run of "<command>". Please review")

  --branch, -b <pr-branch>
    The source branch name to create/push to for the PR.
    (Default: ${branch:-unset})

  --command, -c "<command>"
    The command to execute. Any file changes produced by this command will be
    committed and included in the PR. Required.
    (Default: ${command:-unset})

  --command-path, -m </path/to/which/execute/the/command/>
    The working directory to use when running the command.
    (Default: current working directory)

  --description-command, -d "<command-to-generate-commit-and-pr-message>"
    A command that generates the commit message and PR description. The first
    line becomes the PR title, and lines after the second become the PR body.
    (Default: ${description_command:-unset})

  --dry-run, -D
    Run the command but don't actually commit changes or create/update the PR.
    (Default: ${dry_run:-false})

  --email, -e <git-email>
    The git author email to use when committing changes.
    (Default: ${git_email:-unset})

  --head-branch, -h <head-branch>
    Reuse any self-authored open PR from this branch. This takes priority over
    matching by title when finding existing PRs to update.
    (Default: ${head_branch:-<pr-branch>})

  --labels, -L label1,..,labelN
    Comma-separated list of labels to attach to the PR.
    (Default: ${labels:-unset})

  --missing-labels, -M label1,..,labelN
    Comma-separated list of labels that must be missing on an existing PR.
    If an existing PR from the same source repo and branch has any of these
    labels, the script will exit without performing any action.
    (Default: ${missing_labels:-unset})

  --org, -o <org>
    The GitHub organization name for the PR.
    (Default: ${org:-unset})

  --release-note-none, -R
    Append a "Release note: NONE" block to the PR body.
    (Default: ${release_note_none:-false})

  --repo, -r <repo>
    The GitHub repository name for the PR.
    (Default: ${repo:-unset})

  --repo-path, -p </path/to/git/repository/checkout>
    Path to the local git repository where changes will be committed.
    (Default: current working directory)

  --summary, -s "<summary>"
    The commit message and PR title (if --description-command is not set).
    (Default: "Run <command>")

  --target, -T <target-branch>
    The target branch in the destination repository to merge the PR into.
    (Default: ${targetbranch:-unset})

  --token, -t </path/to/github/token>
    Path to the file containing the GitHub OAuth token.
    (Default: ${token:-unset})

  --user, -l <github-user>
    The GitHub username used as the source fork owner for the PR.
    (Default: ${user:-unset})
EOF
}

die(){ usage >&2; exit 64; }  # EX_USAGE

# Parse command line arguments
cmdline=$(
    set +x

    opts='author:,body:,branch:,command:,command-path:,description-command:'
    opts+=',dry-run,email:,head-branch:,help,labels:,missing-labels:,org:'
    opts+=',release-note-none,repo:,repo-path:,summary:,target:,token:,user:'

    getopt \
        --name="${progname}" \
        --longoptions="${opts}" \
        --options='b:B:c:d:De:h:Hl:L:m:M:n:o:p:r:Rs:t:T:' \
        -- "$@"
) || die

eval set -- "${cmdline}"

# Convert command line arguments to local variables
while [ "$#" -gt 0 ]; do
    case $1 in
        --author              | -n) git_author=$2;           shift;;
        --body                | -B) body=$2;                 shift;;
        --branch              | -b) branch=$2;               shift;;
        --command             | -c) command=$2;              shift;;
        --command-path        | -m) command_path=$2;         shift;;
        --description-command | -d) description_command=$2;  shift;;
        --dry-run             | -D) dry_run='true'                ;;
        --email               | -e) git_email=$2;            shift;;
        --head-branch         | -h) head_branch=$2;          shift;;
        --labels              | -L) labels=$2;               shift;;
        --missing-labels      | -M) missing_labels=$2;       shift;;
        --org                 | -o) org=$2;                  shift;;
        --release-note-none   | -R) release_note_none='true'      ;;
        --repo                | -r) repo=$2;                 shift;;
        --repo-path           | -p) repo_path=$2;            shift;;
        --summary             | -s) summary=$2;              shift;;
        --target              | -T) targetbranch=$2;         shift;;
        --token               | -t) token=$2;                shift;;
        --user                | -l) user=$2;                 shift;;

        --help|-H) usage;  exit;;
        --)        shift; break;;  # End of options
        *)         die;;           # Should not happen
    esac
    shift
done

if [ -z "${command}" ]; then
    die
fi

if [ -n "${missing_labels}" ]; then
    if ! labels-checker \
        --org=kubevirt \
        --repo="${repo}" \
        --author="${user}" \
        --branch-name="${branch}" \
        --ensure-labels-missing="${missing_labels}" \
        --github-token-path="${token}"; then
        echo "skipping PR creation since a PR exists and has one of labels ${missing_labels}"
        exit 0
    fi
fi

if [ -z "${summary}" ]; then
    summary="Run ${command}"
fi

cd "${command_path}"
eval "${command}"

cd "${repo_path}"

if [ -z "$(git status --porcelain)" ]; then
    echo "Nothing changed" >&2
    exit 0
fi

git config user.name "${git_author}"
git config user.email "${git_email}"

if ! git config user.name &>/dev/null && git config user.email &>/dev/null; then
    echo "ERROR: git config user.name, user.email unset. No defaults provided" >&2
    exit 1
fi

if [ -n "${description_command}" ]; then
    summary=$(eval "${description_command}")
    title=$(echo "$summary" | head -1)
    generated_body=$(echo "$summary" | sed '1,2d')
else
    title="$summary"
    generated_body="Automatic run of \"${command}\". Please review"
fi

if [ -z "${body}" ]; then
    body="${generated_body}"
fi

if [ -n "$release_note_none" ]; then
    body+=$'\n\n```release-note\nNONE\n```'
fi

if [ -z "${head_branch}" ]; then
    head_branch="${branch}"
fi

git add -A

fork_url="https://${user}@github.com/${user}/${repo}.git"
if git fetch --depth 1 --no-tags "${fork_url}" "${branch}" 2>/dev/null; then
    local_tree=$(git write-tree)
    remote_tree=$(git rev-parse "FETCH_HEAD^{tree}" 2>/dev/null || true)
    if [ -n "${remote_tree}" ] && [ "${local_tree}" = "${remote_tree}" ]; then
        echo "PR branch ${user}:${branch} already has identical content, skipping" >&2
        exit 0
    fi
fi

if [ -z "$dry_run" ]; then
    git commit -s -m "${summary//[@#]/}"
    git push -f "https://${user}@github.com/${user}/${repo}.git" HEAD:"${branch}"
else
    echo "dry_run: git commit -s -m \"${summary}\""
    echo "dry_run: git push -f \"https://${user}@github.com/${user}/${repo}.git\" HEAD:\"${branch}\""
fi

if [ -z "$dry_run" ]; then
    echo "Creating PR to merge ${user}:${branch} into master..." >&2
    pr-creator \
        --github-token-path="${token}" \
        --org="${org}" --repo="${repo}" --branch="${targetbranch}" \
        --title="${title}" \
        --head-branch="${head_branch}" \
        --body="${body}" \
        --source="${user}":"${branch}" \
        --labels="${labels}" \
        --confirm
else
    pr-creator --help
fi

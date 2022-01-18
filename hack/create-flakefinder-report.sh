#!/usr/bin/env bash

# TODO: integrate this usecase into the go code!

set -xeuo pipefail

default_number_of_prs=20
default_subdir_regex='pull-kubevirt-e2e-k8s-1\.[0-9]+-(sig-.*|operator)'
default_state_of_prs='all'

function usage {
    cat <<EOF
usage: $0 [-n number-of-prs] [-r subdir-regex] [-s state-of-prs]
       $0 [-h]

    creates a flakefinder report over the test results from PRs against kubevirt/kubevirt default branch

    PRs are fetched via GitHub API, see https://docs.github.com/en/rest/reference/pulls for more details

    Arguments:

        -n number-of-prs

            the number of PRs to evaluate (default: $default_number_of_prs)

        -r subdir-regex

            the regular expression to filter jobs with (default: '$default_subdir_regex')

        -s state-of-prs

            filter by state of PRs (default: $default_state_of_prs)

        -h

            display this help text
EOF
}

if ! command -V bazel; then
    echo "bazel is required to run the report creator, see https://bazel.build/"
fi

if ! command -V gh; then
    echo "GitHub cli is required to retrieve PRs, see https://github.com/cli/cli"
fi

number_of_prs=$default_number_of_prs
subdir_regex=$default_subdir_regex
state_of_prs=$default_state_of_prs

while getopts n:r:s:h flag
do
    case "${flag}" in
        n) number_of_prs=${OPTARG};;
        r) subdir_regex=${OPTARG};;
        s) state_of_prs=${OPTARG}
            if ! [[ "$state_of_prs" =~ open|closed|all ]]; then
                usage
                exit 1
            fi
            ;;
        h) usage; exit 0;;
        *) usage; exit 1;;
    esac
done

(
    cd $GH/kubevirt.io/project-infra
    bazel run //robots/cmd/flake-report-creator -- --bucket-name=kubevirt-prow  --sub-dir-regex="$subdir_regex" $(gh api "repos/kubevirt/kubevirt/pulls?per_page=$number_of_prs&base=main&state=$state_of_prs" | jq -r '.[].number | tostring | " --job-data-path=pr-logs/pull/kubevirt_kubevirt/"+.')
)

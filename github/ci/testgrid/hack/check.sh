#!/bin/bash

set -euo pipefail

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source ${BASEDIR}/common.sh

main(){
    generate_config "${@}"

    if git diff --cached --quiet --exit-code; then
        echo "No changes in testgrid config. Aborting no-op bump"
        exit 0
    fi

    run_tests "${@}"
}

main "${@}"

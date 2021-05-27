#!/bin/bash

set -euo pipefail

BASEDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
source ${BASEDIR}/common.sh

main(){
    # make transfigure image pr-creator available in the path
    ln -s /pr-creator /usr/local/bin/pr-creator

    generate_config "${@}"

    run_tests "${@}"
}

main "${@}"

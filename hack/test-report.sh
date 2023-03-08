#!/usr/bin/bash

function run_test_report() {
    podman run -v "$tmp_dir:/tmp:Z" \
            --network host \
            quay.io/kubevirtci/test-report:v20230303-9f7c25ce \
            "$@"
}

tmp_dir="$(mktemp -d)"
if [[ $* =~ ^dequarantine.report ]]; then
    run_test_report --output-file=/tmp/test-report.json "$@"
elif [[ $* =~ ^dequarantine.execute ]]; then
    run_test_report --output-file=/tmp/quarantined_tests.json "$@"
else
    run_test_report --overwrite --outputFile=/tmp/test-report.html "$@"
fi

echo "test-report output written to $tmp_dir: $(ls $tmp_dir)"

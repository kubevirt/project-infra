#!/usr/bin/bash

tmp_dir="$(mktemp -d)"
podman run -v $tmp_dir:/tmp:Z \
        --network host \
        quay.io/kubevirtci/test-report:v20221208-e2f942b1 \
        --overwrite --outputFile=/tmp/report.html "$@"

echo "test-report written to file $tmp_dir/report.html"

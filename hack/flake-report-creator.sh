#!/usr/bin/bash

tmp_dir="$(mktemp -d)"
docker run -v /etc/pki:/etc/pki -v /etc/ssl:/etc/ssl \
        -v "$(dirname $(echo $GOOGLE_APPLICATION_CREDENTIALS)):$(dirname $(echo $GOOGLE_APPLICATION_CREDENTIALS))" \
        -e GOOGLE_APPLICATION_CREDENTIALS="$GOOGLE_APPLICATION_CREDENTIALS" \
        -v "${tmp_dir}:/tmp:Z" \
        --network host \
        quay.io/kubevirtci/flake-report-creator:v20221104-9aec1ec5 \
        --overwrite --outputFile=/tmp/report.html \
        "$@"

echo "$tmp_dir/report.html"

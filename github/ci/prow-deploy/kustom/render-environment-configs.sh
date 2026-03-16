#!/usr/bin/env bash

set -e

OVERLAY="$1"
if [ -z "${OVERLAY}" ]; then
    echo "specify the overlay"
    exit 1
fi

CURRENT_DIR=$(dirname "$0")
BASE_DIR="${CURRENT_DIR}/../files"
if [ ! -d "$BASE_DIR" ]; then
    echo "Cannot find base configs in $BASE_DIR"
    exit 1
fi

OVERLAY_DIR="overlays/${OVERLAY}"
if [ ! -d "$OVERLAY_DIR" ]; then
    echo "Cannot find overlay ${OVERLAY} in ${OVERLAY_DIR}"
    exit 1
fi

CONFIGS=( "config/config.yaml" "plugins/plugins.yaml" "labels/labels.yaml" "mirror/mirror.yaml")

echo "Rendering from base configs at $BASE_DIR"
echo "Using yq_scripts from overlay at $OVERLAY_DIR"

for config in ${CONFIGS[@]}; do
    config_dir=$(dirname $config)
    config_file=$(basename $config)
    mkdir -p "${OVERLAY_DIR}/configs/${config_dir}"

    if [ -f "$OVERLAY_DIR/yq_scripts/$config_file" ]; then
        echo "Applying commands from yq script at $OVERLAY_DIR/yq_scripts/$config_file for $BASE_DIR/$config_file in overlay ${OVERLAY}"

        yq --from-file $OVERLAY_DIR/yq_scripts/$config_file \
          $BASE_DIR/$config_file > $OVERLAY_DIR/configs/$config
    else
        echo "No yq script for $BASE_DIR/$config_file in overlay ${OVERLAY}, copying without changes"
        yq 'true' $BASE_DIR/$config_file >/dev/null
        cp -v $BASE_DIR/$config_file $OVERLAY_DIR/configs/$config
    fi
done

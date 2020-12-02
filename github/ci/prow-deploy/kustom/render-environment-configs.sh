#!/usr/bin/env bash

set -e

YQ_BIN=/tmp/yq
CONFIG_VERSION=${2:-"current"}
BASE_DIR="base/configs/$CONFIG_VERSION"
ENVIRONMENT_DIR="environments/$1"

if [ -z "$1" ]; then
    echo "specify the environment"
    exit 1
fi

CONFIGS=( "config/config.yaml" "plugins/plugins.yaml" "labels/labels.yaml" "docker-mirror/config.yaml"
          "cat-api/api-key")

if [[ ! -d "$BASE_DIR" ]]; then
    echo "Cannot find base configs in $BASE_DIR"
    exit 1
fi

echo "Rendering from base configs at $BASE_DIR with config version \"$CONFIG_VERSION\""

if [[ ! -d "$ENVIRONMENT_DIR" ]]; then
    echo "Cannot find environment $1 in $ENVIRONMENT_DIR"
    exit 1
fi

echo "Using yq_scripts from environment at $ENVIRONMENT_DIR"

for config in ${CONFIGS[@]}; do
    mkdir -p $ENVIRONMENT_DIR/configs/$(dirname $config)
    if [[ -f "$ENVIRONMENT_DIR/yq_scripts/$config" ]]; then
        echo "Apply commands from yq script at $ENVIRONMENT_DIR/yq_scripts/$config for $BASE_DIR/$config in environment $1"
        # yq has two levels of verbosity:
        # - nothing
        # - I'm gonna clog the crap out of your console scrollback buffer for years to come
        # So don't add --verbose unless you're debugging an update script, and even then, good luck.
        ${YQ_BIN} write $BASE_DIR/$config \
          --script $ENVIRONMENT_DIR/yq_scripts/$config >  $ENVIRONMENT_DIR/configs/$config
    else
        # TODO add yaml validation at this step!
        echo "No yq script for $BASE_DIR/$config in environment $1, copying without changes"
        ${YQ_BIN} validate $BASE_DIR/$config
        cp -v $BASE_DIR/$config $ENVIRONMENT_DIR/configs/$config
    fi
done

echo "Rendering completed. Please check the results, yq has no concept of good or bad, it just executes."

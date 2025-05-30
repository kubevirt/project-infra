#!/bin/bash
set -euo pipefail

config_file="$1"

yq -o=json '.presubmits[].[]' "$config_file" | jq -c '.' | while read -r job; do
  name=$(echo "$job" | jq -r '.name')
  run_before_merge=$(echo "$job" | jq -r '.run_before_merge // false')
  env_val=$(echo "$job" | jq -r '.spec.containers[].env[]? | select(.name == "RUN_BEFORE_MERGE") | .value' | head -n1)

  if [[ "$run_before_merge" == "true" ]]; then
    if [[ "$env_val" != "true" ]]; then
      echo "Job '$name': run_before_merge is true but env var is not true"
      exit 1
    fi
  else
    if [[ "$env_val" == "true" ]]; then
      echo "Job '$name': run_before_merge is false/missing but env var is true"
      exit 1
    fi
  fi
done

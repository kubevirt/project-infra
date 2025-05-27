#!/bin/bash
set -euo pipefail

usage() {
  echo "Usage: $0 <presubmits-config-file>" >&2
}

if [[ $# -ne 1 ]]; then
  usage
  exit 2
fi

config_file="$1"

had_error=0

if ! jobs="$(yq -o=json '.presubmits[].[]' "$config_file" | jq -c '.')"; then
  echo "error: failed to parse $config_file" >&2
  exit 1
fi

if [[ -n "$jobs" ]]; then
  while IFS= read -r job; do
    name=$(echo "$job" | jq -r '.name')
    run_before_merge=$(echo "$job" | jq -r '.run_before_merge // false')
    env_count=$(echo "$job" | jq -r '[.spec.containers[].env[]? | select(.name == "RUN_BEFORE_MERGE")] | length')
    env_val=$(echo "$job" | jq -r '.spec.containers[].env[]? | select(.name == "RUN_BEFORE_MERGE") | .value' | head -n1)

    if (( env_count > 1 )); then
      echo "Job '$name': has duplicate RUN_BEFORE_MERGE env vars" >&2
      had_error=1
    fi

    if [[ "$run_before_merge" == "true" ]]; then
      if [[ "$env_val" != "true" ]]; then
        echo "Job '$name': run_before_merge is true but env var is not true" >&2
        had_error=1
      fi
    else
      if [[ "$env_val" == "true" ]]; then
        echo "Job '$name': run_before_merge is false/missing but env var is true" >&2
        had_error=1
      fi
    fi
  done <<< "$jobs"
fi

exit "$had_error"

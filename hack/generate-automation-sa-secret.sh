#!/usr/bin/env bash
#
# This file is part of the KubeVirt project
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Copyright 2022 Red Hat, Inc.
#
#

# based on
# https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/#to-create-additional-api-tokens
# we create a new token for the service account

set -euo pipefail

function usage {
    cat <<EOF
usage: $0 [-D] <service-account-name>

    Generate a new serviceaccount secret

    Options:

        -D  server dry run (see kubectl create --help)

EOF
}

dry_run=
while getopts ":D" opt; do
    case "${opt}" in
        D )
            dry_run='--dry-run=server'
            shift
            ;;
        \? )
            usage
            exit 1
            ;;
    esac
done


if [ ! "$#" -eq 1 ]; then
    usage
    exit 1
fi

serviceaccount_name="$1"

current_context=$(kubectl config current-context)

context=$(echo "$serviceaccount_name" | sed 's/\-[a-z]\+$//')

kubectl config use-context "$context"
kubectl get sa -n default "$serviceaccount_name"

token_name="$serviceaccount_name-token-$(echo $RANDOM | md5sum | head -c 5)"

kubectl create ${dry_run} -f - <<EOF
apiVersion: v1
kind: Secret
metadata:
  name: $token_name
  namespace: default
  annotations:
    kubernetes.io/service-account.name: $serviceaccount_name
type: kubernetes.io/service-account-token
EOF

kubectl config use-context "$current_context"

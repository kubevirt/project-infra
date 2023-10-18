#!/bin/bash

set -eo pipefail

main() {
	local environment=${1}

	bazelisk run //github/ci/services/grafana:$environment.apply

	bazelisk run //github/ci/services/common/k8s/cmd/wait -- -namespace monitoring -selector grafana-deployment -kind deployment

	# Allow some time for the grafana operator to create the required pods & services
	sleep 10

	kubectl apply -n monitoring -f ./github/ci/services/grafana/manifests/ingress.yaml

}

main "${@}"

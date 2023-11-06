#!/bin/bash

set -eo pipefail

main() {
	local environment=${1}

	bazelisk run //github/ci/services/grafana:$environment.apply

	bazelisk run //github/ci/services/common/k8s/cmd/wait -- -namespace monitoring -selector grafana-deployment -kind deployment

	kubectl wait --timeout 60s --for=jsonpath='{.status.stage}'=complete -n monitoring grafana grafana

	kubectl apply -n monitoring -f ./github/ci/services/grafana/manifests/ingress.yaml

}

main "${@}"

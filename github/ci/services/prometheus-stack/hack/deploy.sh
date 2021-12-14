#!/bin/bash

set -eo pipefail

main(){
    local environment=${1}

    bazelisk run //github/ci/services/prometheus-stack:${environment}-crds.apply

    bazelisk run //github/ci/services/common/k8s/cmd/wait -- -selector alertmanagers.monitoring.coreos.com -kind crd

    bazelisk run //github/ci/services/prometheus-stack:${environment}.apply

    bazelisk run //github/ci/services/prometheus-stack:${environment}-k8s-services.apply

    bazelisk run //github/ci/services/common/k8s/cmd/wait -- -namespace monitoring -selector prometheus-stack-kube-prom-operator -kind deployment

    bazelisk run //github/ci/services/common/k8s/cmd/wait -- -namespace monitoring -selector alertmanager-prometheus-stack-kube-prom-alertmanager -kind statefulset

    if [ "${environment}" != "production-e2e-workloads" ]; then
        bazelisk run //github/ci/services/common/k8s/cmd/wait -- -namespace monitoring -selector grafana -kind deployment
    fi

    bazelisk run //github/ci/services/common/k8s/cmd/wait -- -namespace monitoring -selector node-exporter-prometheus-node-exporter -kind daemonset

    bazelisk run //github/ci/services/prometheus-stack:${environment}-service-monitors.apply

    bazelisk run //github/ci/services/prometheus-stack:${environment}-rules.apply
}

main "${@}"

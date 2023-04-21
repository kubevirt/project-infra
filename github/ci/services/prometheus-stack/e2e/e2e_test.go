package e2e

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/check"
)

const (
	testNamespace = "monitoring"
)

func TestPrometheusStackDeployment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "prometheus-stack deployment suite")
}

var _ = Describe("prometheus-stack deployment", func() {
	DescribeTable("should deploy HTTP services",
		func(svcPort, labelSelector, expectedContent, urlPath string) {
			check.HTTPService(testNamespace, svcPort, labelSelector, expectedContent, urlPath)
		},
		Entry("grafana service", "3000", "app.kubernetes.io/name=grafana", "<title>Grafana</title>", ""),
		Entry("prometheus service", "9090", "app=prometheus", "<title>Prometheus Time Series Collection and Processing Server</title>", ""),
		Entry("alertmanager service", "9093", "app=alertmanager", "<title>Alertmanager</title>", ""),
		Entry("node-exporter service", "9100", "app=prometheus-node-exporter", "<title>Node Exporter</title>", ""),
	)
})

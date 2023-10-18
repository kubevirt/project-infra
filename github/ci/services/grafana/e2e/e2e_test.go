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
	RunSpecs(t, "grafana suite")
}

var _ = Describe("grafana deployment", func() {
	DescribeTable("should deploy HTTP services",
		func(svcPort, labelSelector, expectedContent, urlPath string) {
			check.HTTPService(testNamespace, svcPort, labelSelector, expectedContent, urlPath)
		},
		Entry("grafana service", "3000", "app=grafana", "<title>Grafana</title>", ""),
	)
})

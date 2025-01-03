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
	It("should deploy HTTP services", func() {
		check.HTTPService(testNamespace, "3000", "app=grafana", "<title>Grafana</title>", "")
	})
})

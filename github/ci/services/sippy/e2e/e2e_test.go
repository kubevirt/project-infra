package e2e

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/check"
)

const (
	testNamespace = "sippy"
)

func TestSippyDeployment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "sippy deployment suite")
}

var _ = Describe("sippy deployment", func() {
	It("creates a responding HTTP service", func() {
		check.HTTPService(testNamespace, "8080", "app=sippy", "<title>Release CI Health Dashboard</title>", "")
	})
})

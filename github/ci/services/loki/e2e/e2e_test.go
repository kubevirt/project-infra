package e2e

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/check"
)

const (
	testNamespace = "monitoring"
)

func TestLokiDeployment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "loki deployment suite")
}

var _ = Describe("loki deployment", func() {
	It("creates a responding HTTP service", func() {
		check.HTTPService(testNamespace, "3100", "app=loki", "", "")
	})
})

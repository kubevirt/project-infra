package e2e

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/check"
)

const (
	testNamespace = "ci-search"
)

func TestCISearchDeployment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ci-search deployment suite")
}

var _ = Describe("ci-search deployment", func() {
	It("creates a responding HTTP service", func() {
		check.HTTPService(testNamespace, "8080", "app=search", "Search OpenShift CI", "")
	})
})

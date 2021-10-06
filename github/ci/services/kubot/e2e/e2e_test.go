package e2e

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/check"
)

const (
	testNamespace = "kubot"
)

func TestCISearchDeployment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "kubot deployment suite")
}

var _ = Describe("kubot deployment", func() {
	It("creates a responding HTTP health service", func() {
		check.HTTPService(testNamespace, "8080", "app=kubot", "OK", "health")
	})
})

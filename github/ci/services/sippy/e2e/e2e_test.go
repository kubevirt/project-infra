package e2e

import (
	"bufio"
	"fmt"
	"net/http"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes"

	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/client"
	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/portforwarder"
	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/wait"
)

const (
	testNamespace = "sippy"
)

var (
	clientset *kubernetes.Clientset
)

func TestSippyDeployment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "sippy deployment suite")
}

var _ = BeforeSuite(func() {
	var err error

	// initilize clientset
	clientset, err = client.NewClientset()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("sippy deployment", func() {
	It("creates a responding HTTP service", func() {
		localPort := "8080"
		svcPort := "8080"
		ports := []string{fmt.Sprintf("%s:%s", localPort, svcPort)}

		stopChan := make(chan struct{}, 1)
		defer close(stopChan)

		podName := "sippy-0"
		go func() {
			err := portforwarder.New(testNamespace, podName, ports, stopChan)
			if err != nil {
				panic(err)
			}
		}()

		host := "localhost"
		err := wait.ForPortOpen(host, localPort)
		Expect(err).NotTo(HaveOccurred())

		url := fmt.Sprintf("http://%s:%s", host, localPort)
		res, err := http.Get(url)
		Expect(err).NotTo(HaveOccurred())

		defer res.Body.Close()
		scanner := bufio.NewScanner(res.Body)

		expected := "CI Release kubevirt/kubevirt Health Summary"
		found := false
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), expected) {
				found = true
				break
			}
		}
		Expect(found).To(BeTrue())
		Expect(scanner.Err()).NotTo(HaveOccurred())
	})
})

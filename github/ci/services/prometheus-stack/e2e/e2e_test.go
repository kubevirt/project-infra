package e2e

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/client"
	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/portforwarder"
	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/wait"
)

const (
	testNamespace = "monitoring"
)

var (
	clientset *kubernetes.Clientset
)

func TestPrometheusStackDeployment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "prometheus-stack deployment suite")
}

var _ = BeforeSuite(func() {
	var err error

	// initilize clientset
	clientset, err = client.NewClientset()
	Expect(err).NotTo(HaveOccurred())
})

var _ = Describe("prometheus-stack deployment", func() {
	table.DescribeTable("should deploy HTTP services",
		func(svcPort, labelSelector, expectedContent, urlPath string) {
			localPort := "8080"
			ports := []string{fmt.Sprintf("%s:%s", localPort, svcPort)}

			stopChan := make(chan struct{}, 1)
			defer close(stopChan)

			pods, err := clientset.CoreV1().
				Pods(testNamespace).
				List(context.TODO(),
					metav1.ListOptions{
						LabelSelector: labelSelector,
					})
			Expect(err).NotTo(HaveOccurred())
			Expect(len(pods.Items)).To(Equal(1))
			podName := pods.Items[0].Name

			go func() {
				err := portforwarder.New(testNamespace, podName, ports, stopChan)
				if err != nil {
					panic(err)
				}
			}()

			host := "localhost"
			err = wait.ForPortOpen(host, localPort)
			Expect(err).NotTo(HaveOccurred())

			url := fmt.Sprintf("http://%s:%s/%s", host, localPort, urlPath)
			res, err := http.Get(url)
			Expect(err).NotTo(HaveOccurred())

			defer res.Body.Close()
			scanner := bufio.NewScanner(res.Body)

			found := false
			for scanner.Scan() {
				if strings.Contains(scanner.Text(), expectedContent) {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue())
			Expect(scanner.Err()).NotTo(HaveOccurred())

		},
		table.Entry("grafana service", "3000", "app.kubernetes.io/name=grafana", "<title>Grafana</title>", ""),
		table.Entry("prometheus service", "9090", "app=prometheus", "<title>Prometheus Time Series Collection and Processing Server</title>", ""),
		table.Entry("alertmanager service", "9093", "app=alertmanager", "<title>Alertmanager</title>", ""),
		table.Entry("loki service", "3100", "app=loki", "ready", "ready"),
	)
})

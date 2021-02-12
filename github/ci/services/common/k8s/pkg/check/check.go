package check

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strings"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/client"
	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/portforwarder"
	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/wait"
)

var (
	clientset *kubernetes.Clientset
	err       error
)

func HTTPService(testNamespace, svcPort, labelSelector, expectedContent, urlPath string) {
	if clientset == nil {
		clientset, err = client.NewClientset()
		Expect(err).NotTo(HaveOccurred())
	}

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
	Expect(len(pods.Items)).To(BeNumerically(">=", 1))
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
}

package e2e

import (
	"context"
	"testing"

	certmanagerv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/client"
	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/wait"
)

const (
	testNamespace = "cert-manager-test"
)

var (
	clientset *kubernetes.Clientset
	nsSpec    *v1.Namespace
)

func TestCertManagerDeployment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "cert-manager deployment suite")
}

var _ = BeforeSuite(func() {
	var err error

	// initilize clientset
	clientset, err = client.NewClientset()
	Expect(err).NotTo(HaveOccurred())

	nsLabels := map[string]string{"name": testNamespace}
	nsSpec = &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: testNamespace, Labels: nsLabels}}
})

var _ = Describe("cert-manager deployment", func() {
	BeforeEach(func() {
		// create test namespace
		_, err := clientset.CoreV1().Namespaces().Create(
			context.TODO(),
			nsSpec,
			metav1.CreateOptions{},
		)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// remove test namespace
		err := clientset.CoreV1().
			Namespaces().
			Delete(
				context.TODO(),
				testNamespace,
				metav1.DeleteOptions{})
		Expect(err).NotTo(HaveOccurred())

		wait.ForNamespaceDeleted(testNamespace)
	})

	It("creates a self signed certificate", func() {
		name := "test-selfsigned"

		certManagerClientSet, err := client.NewCertManagerClientset()
		Expect(err).NotTo(HaveOccurred())

		// create test resources
		issuer := &certmanagerv1.Issuer{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: certmanagerv1.IssuerSpec{
				IssuerConfig: certmanagerv1.IssuerConfig{
					SelfSigned: &certmanagerv1.SelfSignedIssuer{},
				},
			},
		}
		_, err = certManagerClientSet.CertmanagerV1().Issuers(testNamespace).Create(
			context.TODO(),
			issuer,
			metav1.CreateOptions{},
		)
		Expect(err).NotTo(HaveOccurred())

		certificate := &certmanagerv1.Certificate{
			ObjectMeta: metav1.ObjectMeta{
				Name: name,
			},
			Spec: certmanagerv1.CertificateSpec{
				DNSNames:   []string{"example.com"},
				SecretName: "selfsigned-cert-tls",
				IssuerRef: cmmeta.ObjectReference{
					Name: name,
				},
			},
		}
		_, err = certManagerClientSet.CertmanagerV1().Certificates(testNamespace).Create(
			context.TODO(),
			certificate,
			metav1.CreateOptions{},
		)
		Expect(err).NotTo(HaveOccurred())

		wait.ForCertificateReady(testNamespace, name)
	})

	It("creates an ingress for HTTP01 challenges", func() {
		ingressClassName := "nginx"
		hostname := "prow.kubevirt.io"
		pathPrefix := networkingv1.PathTypePrefix
		ingressSpec := &networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Name: "deck",
				Annotations: map[string]string{
					"cert-manager.io/cluster-issuer": "letsencrypt",
				},
			},
			Spec: networkingv1.IngressSpec{
				IngressClassName: &ingressClassName,
				TLS: []networkingv1.IngressTLS{
					{
						Hosts:      []string{hostname},
						SecretName: "tls-example",
					},
				},
				Rules: []networkingv1.IngressRule{
					{
						Host: hostname,
						IngressRuleValue: networkingv1.IngressRuleValue{
							HTTP: &networkingv1.HTTPIngressRuleValue{
								Paths: []networkingv1.HTTPIngressPath{
									{
										Path:     "/hook",
										PathType: &pathPrefix,
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "hook",
												Port: networkingv1.ServiceBackendPort{
													Number: 8888,
												},
											},
										},
									},
									{
										Path:     "/",
										PathType: &pathPrefix,
										Backend: networkingv1.IngressBackend{
											Service: &networkingv1.IngressServiceBackend{
												Name: "deck",
												Port: networkingv1.ServiceBackendPort{
													Number: 80,
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		_, err := clientset.
			NetworkingV1().
			Ingresses(testNamespace).
			Create(
				context.TODO(),
				ingressSpec,
				metav1.CreateOptions{},
			)
		Expect(err).NotTo(HaveOccurred())

		wait.ForHTTP01IngressCreated(testNamespace, hostname)
	})
})

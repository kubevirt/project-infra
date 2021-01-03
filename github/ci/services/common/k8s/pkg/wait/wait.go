package wait

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	certmanagerv1 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	appsv1 "k8s.io/api/apps/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/client"
)

const poll = 5 * time.Second

type stopChan struct {
	c chan struct{}
	sync.Once
}

func newStopChan() *stopChan {
	return &stopChan{c: make(chan struct{})}
}

func (s *stopChan) closeOnce() {
	s.Do(func() {
		close(s.c)
	})
}

func ForDeploymentReady(namespace, name string) {
	clientset, err := client.NewClientset()
	if err != nil {
		log.Fatalf("Cloud not create clientset %v", err)
	}

	stop := newStopChan()

	watchlist := cache.NewListWatchFromClient(clientset.AppsV1().RESTClient(), "deployments", namespace, fields.Everything())
	_, controller := cache.NewInformer(watchlist, &appsv1.Deployment{}, poll, cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(o, n interface{}) {
			newDeployment := n.(*appsv1.Deployment)

			if newDeployment.Name != name {
				return
			}
			if isDeploymentReady(newDeployment) {
				stop.closeOnce()
				return
			}
		},
	})
	controller.Run(stop.c)
}

func ForCertificateReady(namespace, name string) {
	clientset, err := client.NewCertManagerClientset()
	if err != nil {
		log.Fatalf("Cloud not create clientset %v", err)
	}

	stop := newStopChan()

	watchlist := cache.NewListWatchFromClient(clientset.CertmanagerV1().RESTClient(), "certificates", namespace, fields.Everything())
	_, controller := cache.NewInformer(watchlist, &certmanagerv1.Certificate{}, poll, cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(o, n interface{}) {
			newCertificate := n.(*certmanagerv1.Certificate)

			if newCertificate.Name != name {
				return
			}
			if isCertificateReady(newCertificate) {
				stop.closeOnce()
				return
			}
		},
	})
	controller.Run(stop.c)
}

func isDeploymentReady(deployment *appsv1.Deployment) bool {
	return deployment.Status.UpdatedReplicas == *deployment.Spec.Replicas &&
		deployment.Status.Replicas == *deployment.Spec.Replicas &&
		deployment.Status.AvailableReplicas == *deployment.Spec.Replicas &&
		deployment.Status.ObservedGeneration >= deployment.Generation
}

func isCertificateReady(certificate *certmanagerv1.Certificate) bool {
	for _, condition := range certificate.Status.Conditions {
		if condition.Type == certmanagerv1.CertificateConditionReady &&
			condition.Status == cmmeta.ConditionTrue {
			return true
		}
	}
	return false
}

func ForNamespaceDeleted(namespace string) {
	clientset, err := client.NewClientset()
	if err != nil {
		log.Fatalf("Cloud not create clientset %v", err)
	}

	watcher, err := clientset.
		CoreV1().
		Namespaces().
		Watch(
			context.TODO(),
			metav1.ListOptions{
				LabelSelector: fmt.Sprintf("name=%s", namespace),
			})
	if err != nil {
		log.Fatalf("Cloud not watch namespace %v", err)
	}

	for {
		select {
		case res := <-watcher.ResultChan():
			if res.Type == watch.Deleted {
				return
			}
		case <-time.After(3 * time.Minute):
			log.Fatalf("Namespace %s not deleted in time", namespace)
		}
	}
}

func ForHTTP01IngressCreated(namespace, hostname string) {
	clientset, err := client.NewClientset()
	if err != nil {
		log.Fatalf("Cloud not create clientset %v", err)
	}

	stop := newStopChan()

	watchlist := cache.NewListWatchFromClient(clientset.NetworkingV1().RESTClient(), "ingresses", namespace, fields.Everything())
	_, controller := cache.NewInformer(watchlist, &networkingv1.Ingress{}, poll, cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(o, n interface{}) {
			newIngress := n.(*networkingv1.Ingress)

			found := false
			for labelKey, labelValue := range newIngress.Labels {
				if labelKey == "acme.cert-manager.io/http01-solver" ||
					labelValue == "true" {
					found = true
				}
			}
			if !found {
				return
			}
			if isHTTP01Ingress(newIngress, hostname) {
				stop.closeOnce()
				return
			}
		},
	})
	controller.Run(stop.c)
}

func isHTTP01Ingress(ingress *networkingv1.Ingress, hostname string) bool {
	for _, rule := range ingress.Spec.Rules {
		if rule.Host != hostname {
			continue
		}
		for _, ingressPath := range rule.IngressRuleValue.HTTP.Paths {
			if strings.Contains(ingressPath.Path, ".well-known/acme-challenge/") {
				return true
			}
		}
	}
	return false
}

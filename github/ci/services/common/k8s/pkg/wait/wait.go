package wait

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
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
const timeout = 200 * time.Second

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
		log.Fatalf("Could not create clientset %v", err)
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
	waitForControllerWithTimeout(controller, stop, name, namespace)
}

func ForCertificateReady(namespace, name string) {
	clientset, err := client.NewCertManagerClientset()
	if err != nil {
		log.Fatalf("Could not create clientset %v", err)
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
	waitForControllerWithTimeout(controller, stop, name, namespace)
}

func ForNamespaceDeleted(namespace string) {
	clientset, err := client.NewClientset()
	if err != nil {
		log.Fatalf("Could not create clientset %v", err)
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
		log.Fatalf("Could not watch namespace %v", err)
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
		log.Fatalf("Could not create clientset %v", err)
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
	waitForControllerWithTimeout(controller, stop, hostname, namespace)
}

func ForStatefulsetReady(namespace, name string) {
	clientset, err := client.NewClientset()
	if err != nil {
		log.Fatalf("Could not create clientset %v", err)
	}

	stop := newStopChan()

	watchlist := cache.NewListWatchFromClient(clientset.AppsV1().RESTClient(), "statefulsets", namespace, fields.Everything())
	_, controller := cache.NewInformer(watchlist, &appsv1.StatefulSet{}, poll, cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(o, n interface{}) {
			newStatefulset := n.(*appsv1.StatefulSet)

			if newStatefulset.Name != name {
				return
			}
			if isStatefulsetReady(newStatefulset) {
				stop.closeOnce()
				return
			}
		},
	})
	waitForControllerWithTimeout(controller, stop, name, namespace)
}

func ForPortOpen(host, port string) error {
	timeout := time.After(20 * time.Second)
	tick := time.Tick(1 * time.Second)

	for {
		select {
		case <-timeout:
			return errors.New(fmt.Sprintf("Port %s was not open in time", port))
		case <-tick:
			conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), time.Second)
			if err != nil {
				return err
			}
			if conn != nil {
				defer conn.Close()
				return nil
			}
		}
	}
}

func ForDaemonsetReady(namespace, name string) {
	clientset, err := client.NewClientset()
	if err != nil {
		log.Fatalf("Could not create clientset %v", err)
	}

	stop := newStopChan()

	watchlist := cache.NewListWatchFromClient(clientset.AppsV1().RESTClient(), "daemonsets", namespace, fields.Everything())
	_, controller := cache.NewInformer(watchlist, &appsv1.DaemonSet{}, poll, cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(o, n interface{}) {
			newResource := n.(*appsv1.DaemonSet)

			if newResource.Name != name {
				return
			}
			if isDaemonsetReady(newResource) {
				stop.closeOnce()
				return
			}
		},
	})
	waitForControllerWithTimeout(controller, stop, name, namespace)
}

func ForCRDCreated(name string) {
	client, err := client.NewApiextensionsV1Client()
	if err != nil {
		log.Fatalf("Could not create dynamic client %v", err)
	}

	crdsIface := client.CustomResourceDefinitions()
	timeout := time.After(30 * time.Second)
	tick := time.Tick(5 * time.Second)
	for {
		select {
		case <-timeout:
			log.Fatalf("CRD %q not created in time\n", name)
		case <-tick:
			crds, err := crdsIface.List(context.TODO(),
				metav1.ListOptions{
					FieldSelector: fmt.Sprintf("metadata.name=%s", name),
				})
			if err != nil {
				log.Printf("Error querying CRDs: %v", err)
			}
			if len(crds.Items) == 1 && crds.Items[0].Name == name {
				log.Printf("CRD %q ready\n", name)
				return
			}
			log.Printf("Waiting CRD %q to be created...\n", name)
		}
	}
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

func isStatefulsetReady(statefulset *appsv1.StatefulSet) bool {
	return statefulset.Status.UpdatedReplicas == *statefulset.Spec.Replicas &&
		statefulset.Status.Replicas == *statefulset.Spec.Replicas &&
		statefulset.Status.ReadyReplicas == *statefulset.Spec.Replicas &&
		statefulset.Status.ObservedGeneration >= statefulset.Generation
}

func isDaemonsetReady(resource *appsv1.DaemonSet) bool {
	return resource.Status.CurrentNumberScheduled == resource.Status.DesiredNumberScheduled &&
		resource.Status.ObservedGeneration >= resource.Generation
}

func waitForControllerWithTimeout(controller cache.Controller, stop *stopChan, name, namespace string) {
	go controller.Run(stop.c)

	timeoutCh := time.After(timeout)
	tickCh := time.Tick(poll)
	for {
		select {
		case <-timeoutCh:
			stop.closeOnce()
			log.Fatalf("Resource %q in namespace %q not ready in time\n", name, namespace)
		case <-tickCh:
			log.Printf("Waiting for resource %q to be ready in namespace %q...\n", name, namespace)
		case <-stop.c:
			log.Printf("Resource %q in namespace %q is ready\n", name, namespace)
			return
		}
	}
}

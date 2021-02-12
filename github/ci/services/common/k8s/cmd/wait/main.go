package main

import (
	"flag"
	"log"

	"kubevirt.io/project-infra/github/ci/services/common/k8s/pkg/wait"
)

func main() {
	namespace := flag.String("namespace", "default", "namespace where the resources live")
	kind := flag.String("kind", "deployment", "which kind of resource to wait for")
	selector := flag.String("selector", "", "label selector")

	flag.Parse()

	if *selector == "" {
		log.Fatalf("Please specify a selector with -selector")
	}

	switch *kind {
	case "deployment":
		wait.ForDeploymentReady(*namespace, *selector)
	case "statefulset":
		wait.ForStatefulsetReady(*namespace, *selector)
	case "daemonset":
		wait.ForDaemonsetReady(*namespace, *selector)
	case "crd":
		wait.ForCRDCreated(*selector)

	default:
		log.Fatalf("Unknown resource kind, %q", *kind)
	}
}

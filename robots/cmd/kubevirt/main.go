package main

import (
	"os"

	"kubevirt.io/project-infra/robots/pkg/kubevirt"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/log"
)

func main() {
	if err := kubevirt.Execute(); err != nil {
		log.Log().Fatal(err)
	}
	os.Exit(0)
}

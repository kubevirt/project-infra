package main

import (
	"kubevirt.io/project-infra/pkg/kubevirt"
	"kubevirt.io/project-infra/pkg/kubevirt/log"
	"os"
)

func main() {
	if err := kubevirt.Execute(); err != nil {
		log.Log().Fatal(err)
	}
	os.Exit(0)
}

package main

import (
	"os"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/cmd"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/log"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Log().Fatal(err)
	}
	os.Exit(0)
}

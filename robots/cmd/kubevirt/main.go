package main

import (
	"kubevirt.io/project-infra/robots/pkg/kubevirt/cmd"
	"os"
)

func main() {
	cmd.Execute()
	os.Exit(0)
}

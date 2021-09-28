package main

import (
	"os"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/cmd"
)

func main() {
	cmd.Execute()
	os.Exit(0)
}

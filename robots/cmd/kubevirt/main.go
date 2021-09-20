package main

import (
	"kubevirt.io/project-infra/robots/pkg/kubevirt"
	"os"
)

func main() {
	kubevirt.Execute()
	os.Exit(0)
}

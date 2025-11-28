package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	"kubevirt.io/project-infra/robots/cmd/flake-report-creator/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}

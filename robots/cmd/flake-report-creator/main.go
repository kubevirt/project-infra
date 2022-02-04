package main

import (
	log "github.com/sirupsen/logrus"
	"kubevirt.io/project-infra/robots/cmd/flake-report-creator/cmd"
	"os"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}

package main

import (
	"kubevirt.io/project-infra/robots/cmd/flake-report-creator/cmd"
	"log"
	"os"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}

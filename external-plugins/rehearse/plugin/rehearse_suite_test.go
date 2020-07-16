package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestRehearse(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rehearse Suite")
}

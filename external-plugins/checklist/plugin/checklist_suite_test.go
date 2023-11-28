package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestChecklist(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Checklist Suite")
}

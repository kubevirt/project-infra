package main_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestFlakefinder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Flakefinder Suite")
}

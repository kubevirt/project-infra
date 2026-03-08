package server

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestEventsServer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Events Server Suite")
}

package plugin

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRehearse(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rehearse Suite")
}

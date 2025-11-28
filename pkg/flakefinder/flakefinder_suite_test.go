package flakefinder_test

import (
	"flag"
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type options struct {
	printTestOutput bool
}

var testOptions = options{}

func TestMain(m *testing.M) {
	flag.BoolVar(&testOptions.printTestOutput, "print_test_output", false, "Whether test output should be printed via logger")
	flag.Parse()
	os.Exit(m.Run())
}

func TestFlakefinder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Flakefinder Pkg Suite")
}

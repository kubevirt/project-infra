package main

import (
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestPhasedMain(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Phased main suite")
}

var _ = Describe("reconciliation options", func() {
	DescribeTable("parses the interval", func(args []string, expected time.Duration) {
		original := os.Args
		DeferCleanup(func() { os.Args = original })
		os.Args = append([]string{"phased"}, args...)
		opts := gatherOptions()
		Expect(opts.reconcileInterval).To(Equal(expected))
	}, Entry("default", []string{}, 10*time.Minute), Entry("startup only", []string{"--reconcile-interval=0"}, time.Duration(0)), Entry("custom", []string{"--reconcile-interval=2m"}, 2*time.Minute))

	It("rejects negative intervals and accepts startup-only mode", func() {
		opts := &options{prowConfigPath: "config", jobsConfigBase: "jobs", cacheDir: "cache", prowLocation: "https://example.org", reconcileInterval: -time.Second}
		Expect(opts.validate()).To(MatchError(ContainSubstring("reconcile-interval must not be negative")))
		opts.reconcileInterval = 0
		Expect(opts.validate()).To(Succeed())
	})

	It("retains validation of required options", func() {
		opts := &options{}
		Expect(opts.validate()).To(MatchError(ContainSubstring("prow-config-path can't be empty")))
	})
})

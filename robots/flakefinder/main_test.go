package main_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "kubevirt.io/project-infra/robots/flakefinder"
)

var _ = Describe("main.go", func() {

	RegisterFailHandler(Fail)

	reportTime, e := time.Parse("2006-01-02T15:04:05", "2019-08-23T03:27:01")
	Expect(e).ToNot(HaveOccurred())

	h24, e := time.ParseDuration("24h")
	Expect(e).ToNot(HaveOccurred())

	When("creating GH queries", func() {

		It("does not fail for 24", func() {
			query := MakeQuery("blah", h24, reportTime)
			Expect(query).To(BeEquivalentTo("blah merged:>=2019-08-22T00:00:00Z"))
		})

	})
})

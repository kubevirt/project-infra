package main

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/project-infra/pkg/flakefinder"
)

var _ = Describe("main.go", func() {

	When("Setting up output path", func() {

		BeforeEach(func() {
			ReportOutputPath = flakefinder.ReportsPath
		})

		It("has default path", func() {
			options := options{}
			Expect(BuildReportOutputPath(options)).To(BeEquivalentTo("reports/flakefinder"))
		})

		It("has preview if option enabled", func() {
			options := options{isPreview: true}
			Expect(BuildReportOutputPath(options)).To(BeEquivalentTo("reports/flakefinder/preview"))
		})

		It("has child branch", func() {
			options := options{reportOutputChildPath: "master"}
			Expect(BuildReportOutputPath(options)).To(BeEquivalentTo("reports/flakefinder/master"))
		})

		It("has preview and child branch", func() {
			options := options{isPreview: true, reportOutputChildPath: "master"}
			Expect(BuildReportOutputPath(options)).To(BeEquivalentTo("reports/flakefinder/preview/master"))
		})

	})

})

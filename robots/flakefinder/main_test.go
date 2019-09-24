package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "kubevirt.io/project-infra/robots/flakefinder"
)

var _ = Describe("main.go", func() {

	When("Setting up output path", func() {

		BeforeEach(func() {
			ReportOutputPath = ReportsPath
		})

		It("has default path", func() {
			options := Options{}
			Expect(BuildReportOutputPath(options)).To(BeEquivalentTo("reports/flakefinder"))
		})

		It("has preview if option enabled", func() {
			options := Options{IsPreview: true}
			Expect(BuildReportOutputPath(options)).To(BeEquivalentTo("reports/flakefinder/preview"))
		})

		It("has child branch", func() {
			options := Options{ReportOutputChildPath: "master"}
			Expect(BuildReportOutputPath(options)).To(BeEquivalentTo("reports/flakefinder/master"))
		})

		It("has preview and child branch", func() {
			options := Options{IsPreview: true, ReportOutputChildPath: "master"}
			Expect(BuildReportOutputPath(options)).To(BeEquivalentTo("reports/flakefinder/preview/master"))
		})

	})

})

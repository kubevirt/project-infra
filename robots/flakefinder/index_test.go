package main_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "kubevirt.io/project-infra/robots/flakefinder"
)

var _ = Describe("Index", func() {

	When("filtering report objects for index", func() {

		It("filters non report objects", func() {
			reportDirGcsObjects := []string{
				"dunnoWhatFileThisis",
				"whatever-2019-08-22.html",
				"flakefinder-2019-07-24.html",
				"flakefinder-2019-08-22.html",
				"flakefinder-2019-07-17.html",
				"flakefinder-2019-08-22.pdf",
				"flakefinder-2019-07-25.html",
				"thisOtherShouldBeLeftOutAlso",
			}
			reportItemsForIndexPage := FilterReportItemsForIndexPage(reportDirGcsObjects)
			Expect(reportItemsForIndexPage).To(BeEquivalentTo([]string{
				"flakefinder-2019-08-22.html",
				"flakefinder-2019-07-25.html",
				"flakefinder-2019-07-24.html",
				"flakefinder-2019-07-17.html",
			}))
		})

		It("includes different reports", func() {
			reportDirGcsObjects := []string{
				"dunnoWhatFileThisis",
				"flakefinder-2019-08-24-024h.html",
				"flakefinder-2019-07-24-168h.html",
				"flakefinder-2019-08-24-672h.html",
				"flakefinder-2019-08-22-024h.html",
				"whatever-2019-08-22.html",
				"flakefinder-2019-07-17-168h.html",
				"flakefinder-2019-08-23-024h.html",
				"flakefinder-2019-08-24-168h.html",
				"thisOtherShouldBeLeftOutAlso",
			}
			reportItemsForIndexPage := FilterReportItemsForIndexPage(reportDirGcsObjects)
			Expect(reportItemsForIndexPage).To(BeEquivalentTo([]string{
				"flakefinder-2019-08-22.html",
				"flakefinder-2019-07-25.html",
				"flakefinder-2019-07-24.html",
				"flakefinder-2019-07-17.html",
			}))
		})

	})

})

package main_test

import (
	"bytes"
	"fmt"
	"log"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "kubevirt.io/project-infra/robots/flakefinder"
)

var _ = Describe("index.go", func() {

	reportDataFiles := []string{
		"flakefinder-2019-08-24-672h.html",
		"flakefinder-2019-08-24-168h.html",
		"flakefinder-2019-08-24-024h.html",
		"flakefinder-2019-08-23-024h.html",
		"flakefinder-2019-08-22-024h.html",
		"flakefinder-2019-07-24-168h.html",
		"flakefinder-2019-07-17-168h.html",
	}

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
			Expect(reportItemsForIndexPage).To(BeEquivalentTo(reportDataFiles))
		})

	})

	When("writing the index page", func() {

		var htmlIndex string

		prepareBuffer := func(org, repo string) {
			buffer := bytes.Buffer{}
			err := WriteReportIndexPage(reportDataFiles, &buffer, org, repo)
			Expect(err).ToNot(HaveOccurred())
			htmlIndex = buffer.String()
			if testOptions.printTestOutput {
				logger := log.New(os.Stdout, "index_test.go:", log.Flags())
				logger.Println(htmlIndex)
			}
		}

		It("uses all report items", func() {
			prepareBuffer(Org, Repo)

			Expect(htmlIndex).To(ContainSubstring("flakefinder-2019-08-24-672h.html"))
			Expect(htmlIndex).To(ContainSubstring("flakefinder-2019-08-24-168h.html"))
			Expect(htmlIndex).To(ContainSubstring("flakefinder-2019-08-24-024h.html"))
			Expect(htmlIndex).To(ContainSubstring("flakefinder-2019-08-23-024h.html"))
			Expect(htmlIndex).To(ContainSubstring("flakefinder-2019-08-22-024h.html"))
			Expect(htmlIndex).To(ContainSubstring("flakefinder-2019-07-24-168h.html"))
			Expect(htmlIndex).To(ContainSubstring("flakefinder-2019-07-17-168h.html"))
		})

		It("contains the merged duration spans as headers", func() {
			prepareBuffer(Org, Repo)

			Expect(htmlIndex).To(ContainSubstring("<th>672h</th>"))
			Expect(htmlIndex).To(ContainSubstring("<th>024h</th>"))
			Expect(htmlIndex).To(ContainSubstring("<th>168h</th>"))
		})

		DescribeTable("contains the org and repo in title", func(org, repo string) {
			prepareBuffer(org, repo)

			Expect(htmlIndex).To(ContainSubstring(fmt.Sprintf("<title>%s/%s", org, repo)))
		},
			Entry("contains kubevirt/kubevirt", "kubevirt", "kubevirt"),
			Entry("contains kubevirt/containerized-data-importer", "kubevirt", "containerized-data-importer"),
			Entry("contains test/blah", "test", "blah"),
		)

	})

	When("preparing data for index page", func() {

		It("puts it into a map per duration", func() {
			indexParams := PrepareDataForTemplate(reportDataFiles, Org, Repo)
			Expect(indexParams.Reports).To(BeEquivalentTo([]ReportFilesRow{
				{
					Date: "2019-08-24",
					ReportFiles: map[ReportFileMergedDuration]string{
						"672h": "flakefinder-2019-08-24-672h.html",
						"168h": "flakefinder-2019-08-24-168h.html",
						"024h": "flakefinder-2019-08-24-024h.html",
					},
				},
				{
					Date: "2019-08-23",
					ReportFiles: map[ReportFileMergedDuration]string{
						"672h": "",
						"168h": "",
						"024h": "flakefinder-2019-08-23-024h.html",
					},
				},
				{
					Date: "2019-08-22",
					ReportFiles: map[ReportFileMergedDuration]string{
						"672h": "",
						"168h": "",
						"024h": "flakefinder-2019-08-22-024h.html",
					},
				},
				{
					Date: "2019-07-24",
					ReportFiles: map[ReportFileMergedDuration]string{
						"672h": "",
						"168h": "flakefinder-2019-07-24-168h.html",
						"024h": "",
					},
				},
				{
					Date: "2019-07-17",
					ReportFiles: map[ReportFileMergedDuration]string{
						"672h": "",
						"168h": "flakefinder-2019-07-17-168h.html",
						"024h": "",
					},
				},
			}))
		})

		It("is backwards compatible", func() {

			mixedDataFiles := []string{
				"flakefinder-2019-08-24-672h.html",
				"flakefinder-2019-08-24-024h.html",
				"flakefinder-2019-07-24.html",
				"flakefinder-2019-07-17.html",
			}

			indexParams := PrepareDataForTemplate(mixedDataFiles, Org, Repo)
			Expect(indexParams.Reports).To(BeEquivalentTo([]ReportFilesRow{
				{
					Date: "2019-08-24",
					ReportFiles: map[ReportFileMergedDuration]string{
						"672h": "flakefinder-2019-08-24-672h.html",
						"168h": "",
						"024h": "flakefinder-2019-08-24-024h.html",
					},
				},
				{
					Date: "2019-07-24",
					ReportFiles: map[ReportFileMergedDuration]string{
						"672h": "",
						"168h": "flakefinder-2019-07-24.html",
						"024h": "",
					},
				},
				{
					Date: "2019-07-17",
					ReportFiles: map[ReportFileMergedDuration]string{
						"672h": "",
						"168h": "flakefinder-2019-07-17.html",
						"024h": "",
					},
				},
			}))
		})

	})

})

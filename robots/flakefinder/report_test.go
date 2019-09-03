package main_test

import (
	"bytes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"log"
	"os"
	"time"

	. "kubevirt.io/project-infra/robots/flakefinder"
)

var _ = Describe("report.go", func() {

	RegisterFailHandler(Fail)

	reportTime, e := time.Parse("2006-01-02", "2019-08-23")
	Expect(e).ToNot(HaveOccurred())

	When("creates filename with date and merged as hours", func() {

		It("creates a filename for week", func() {
			fileName := CreateReportFileName(reportTime, 24*7*time.Hour)
			Expect(fileName).To(BeEquivalentTo("flakefinder-2019-08-23-168h.html"))
		})

		It("creates a filename for day", func() {
			fileName := CreateReportFileName(reportTime, 24*time.Hour)
			Expect(fileName).To(BeEquivalentTo("flakefinder-2019-08-23-024h.html"))
		})

	})

	When("rendering report data", func() {

		buffer := bytes.Buffer{}
		parameters := Params{Data: map[string]map[string]*Details{
			"t1": map[string]*Details{"a": &Details{Failed: 4, Succeeded: 1, Skipped: 2, Severity: "red", Jobs: []*Job{}}},
		}, Headers: []string{"a", "b", "c"}, Tests: []string{"t1", "t2", "t3"}, Date: "2019-08-23"}
		WriteReportToOutput(&buffer, parameters)

		logger := log.New(os.Stdout, "report.go:", log.Flags())
		logger.Printf(buffer.String())

		It("outputs something", func() {
			Expect(buffer.String()).ToNot(BeEmpty())
		})

		It("has rows", func() {
			Expect(buffer.String()).To(ContainSubstring("<td>t1</td>"))
			Expect(buffer.String()).To(ContainSubstring("<td>t2</td>"))
			Expect(buffer.String()).To(ContainSubstring("<td>t3</td>"))
		})

		It("has columns", func() {
			Expect(buffer.String()).To(ContainSubstring("<td>a</td>"))
			Expect(buffer.String()).To(ContainSubstring("<td>b</td>"))
			Expect(buffer.String()).To(ContainSubstring("<td>c</td>"))
		})

		It("has one filled test cell", func() {
			Expect(buffer.String()).To(ContainSubstring("<td class=\"red center\">"))
			Expect(buffer.String()).To(MatchRegexp("(?s)4.*1.*2"))
		})

		It("contains the date", func() {
			Expect(buffer.String()).To(ContainSubstring("2019-08-23"))
		})

	})

})

package main_test

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"time"

	"kubevirt.io/project-infra/pkg/flakefinder"
	"kubevirt.io/project-infra/pkg/validation"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "kubevirt.io/project-infra/robots/flakefinder"
)

var _ = Describe("report.go", func() {

	RegisterFailHandler(Fail)

	reportTime, e := time.Parse("2006-01-02", "2019-08-23")
	Expect(e).ToNot(HaveOccurred())

	When("creates filename with date and merged as hours", func() {

		It("creates a filename for week", func() {
			fileName := CreateReportFileNameWithEnding(reportTime, 24*7*time.Hour, "html")
			Expect(fileName).To(BeEquivalentTo("flakefinder-2019-08-23-168h.html"))
		})

		It("creates a filename for day", func() {
			fileName := CreateReportFileNameWithEnding(reportTime, 24*time.Hour, "html")
			Expect(fileName).To(BeEquivalentTo("flakefinder-2019-08-23-024h.html"))
		})

	})

	const (
		testName1 = "[release-blocker][Serial][test_id:4217]t1"
		testName2 = "[QUARANTINE][test_id:1742]t2"
		testName3 = "[sig-compute][some-unimportant-label][some-label][some-label][some-label][some-label][some-label][some-label][some-label][some-label][some-label][some-label]t3"
		jobNameA  = "a"
		jobNameB  = "b"
		jobNameC  = "c"
		commitID1 = "d66cb1c"
	)

	When("rendering report data", func() {

		var buffer bytes.Buffer

		prepareBuffer := func(parameters flakefinder.Params) {
			buffer = bytes.Buffer{}
			err := flakefinder.WriteTemplateToOutput(ReportTemplate, parameters, &buffer)
			Expect(err).ToNot(HaveOccurred())
			if testOptions.printTestOutput {
				logger := log.New(os.Stdout, "report_test.go:", log.Flags())
				logger.Println(buffer.String())
			}
		}

		prepareWithDefaultParams := func() {
			parameters := flakefinder.Params{
				Data: map[string]map[string]*flakefinder.Details{
					testName1: {
						jobNameA: &flakefinder.Details{
							Failed:           4,
							Succeeded:        1,
							Skipped:          2,
							Severity:         "red",
							NonDeterministic: true,
							Jobs: []*flakefinder.Job{
								{
									BuildNumber: 1,
									Severity:    "red",
									CommitID:    commitID1,
								},
								{
									BuildNumber: 2,
									Severity:    "green",
									CommitID:    commitID1,
								},
							}},
					},
					testName2: {
						jobNameB: &flakefinder.Details{
							Failed:    1,
							Succeeded: 5,
							Skipped:   1,
							Severity:  "red",
							Jobs:      []*flakefinder.Job{}},
					},
					testName3: {
						jobNameC: &flakefinder.Details{
							Failed:    9,
							Succeeded: 3,
							Skipped:   1,
							Severity:  "red",
							Jobs:      []*flakefinder.Job{}},
					},
				},
				Headers: []string{jobNameA, jobNameB, jobNameC},
				Tests:   []string{testName1, testName2, testName3},
				TestAttributes: map[string]flakefinder.TestAttributes{
					testName1: flakefinder.NewTestAttributes(testName1),
					testName2: flakefinder.NewTestAttributes(testName2),
					testName3: flakefinder.NewTestAttributes(testName3),
				},
				BareTestNames: map[string]string{
					testName1: flakefinder.GetBareTestName(testName1),
					testName2: flakefinder.GetBareTestName(testName2),
					testName3: flakefinder.GetBareTestName(testName3),
				},
				EndOfReport: "2019-08-23",
				Org:         Org,
				Repo:        Repo,
				PrNumbers:   []int{17, 42},
			}

			prepareBuffer(parameters)
		}

		prepareWithNoFailingTests := func() {
			parameters := flakefinder.Params{Data: map[string]map[string]*flakefinder.Details{},
				Headers: []string{}, Tests: []string{}, EndOfReport: "2019-08-23",
				Org: Org, Repo: Repo,
				PrNumbers: []int{17, 42},
			}

			prepareBuffer(parameters)
		}

		It("outputs something", func() {
			prepareWithDefaultParams()
			Expect(buffer.String()).ToNot(BeEmpty())
		})

		It("has rows", func() {
			prepareWithDefaultParams()
			Expect(buffer.String()).To(ContainSubstring(flakefinder.NewTestAttributes(testName1)[0].Name))
			Expect(buffer.String()).To(ContainSubstring("<div class=\"testAttribute\""))
			Expect(buffer.String()).To(ContainSubstring(testName1))
			Expect(buffer.String()).To(ContainSubstring(testName2))
			Expect(buffer.String()).To(ContainSubstring(testName3))
		})

		It("has columns", func() {
			prepareWithDefaultParams()
			Expect(buffer.String()).To(ContainSubstring(fmt.Sprintf("<td>%s</td>", jobNameA)))
			Expect(buffer.String()).To(ContainSubstring(fmt.Sprintf("<td>%s</td>", jobNameB)))
			Expect(buffer.String()).To(ContainSubstring(fmt.Sprintf("<td>%s</td>", jobNameC)))
		})

		It("has one filled test cell", func() {
			prepareWithDefaultParams()
			Expect(buffer.String()).To(ContainSubstring("<td class=\"red nondeterministic center\""))
			Expect(buffer.String()).To(MatchRegexp("(?s)4.*1.*2"))
		})

		It("contains the date", func() {
			prepareWithDefaultParams()
			Expect(buffer.String()).To(ContainSubstring("2019-08-23"))
		})

		It("contains the pr ids", func() {
			prepareWithDefaultParams()
			Expect(buffer.String()).To(ContainSubstring("#17"))
			Expect(buffer.String()).To(ContainSubstring("#42"))
		})

		It("creates valid html with default params", func() {
			prepareWithDefaultParams()
			Expect(validation.HTMLValidator{}.IsValid(buffer.Bytes())).To(BeNil())
		})

		It("shows no errors if no failing tests", func() {
			prepareWithNoFailingTests()
			Expect(buffer.String()).To(ContainSubstring("No failing tests!"))
		})

		It("shows pr ids if no failing tests", func() {
			prepareWithNoFailingTests()
			Expect(buffer.String()).To(ContainSubstring("#17"))
			Expect(buffer.String()).To(ContainSubstring("#42"))
		})

		DescribeTable("title contains repo and org", func(org, repo string) {
			parameters := flakefinder.Params{Data: map[string]map[string]*flakefinder.Details{
				"t1": {jobNameA: &flakefinder.Details{Failed: 4, Succeeded: 1, Skipped: 2, Severity: "red", Jobs: []*flakefinder.Job{}}},
			}, Headers: []string{jobNameA, "b", "c"}, Tests: []string{"t1", "t2", testName3}, EndOfReport: "2019-08-23", Org: org, Repo: repo}

			prepareBuffer(parameters)

			Expect(buffer.String()).To(ContainSubstring(fmt.Sprintf("<title>%s/%s", org, repo)))
		},
			Entry("is kubevirt/kubevirt", "kubevirt", "kubevirt"),
			Entry("is kubevirt/containerized-data-importer", "kubevirt", "containerized-data-importer"),
			Entry("is test/blah", "test", "blah"),
		)

		DescribeTable("prow link contains repo and org", func(org, repo string) {
			parameters := flakefinder.Params{Data: map[string]map[string]*flakefinder.Details{
				"t1": {jobNameA: &flakefinder.Details{Failed: 4, Succeeded: 1, Skipped: 2, Severity: "red", Jobs: []*flakefinder.Job{
					{BuildNumber: 1742, Severity: "red", PR: 1427, Job: "testblah"},
				}}},
			}, Headers: []string{jobNameA, "b", "c"}, Tests: []string{"t1", "t2", testName3}, EndOfReport: "2019-08-23", Org: org, Repo: repo}

			prepareBuffer(parameters)

			Expect(buffer.String()).To(ContainSubstring(fmt.Sprintf("pr-logs/pull/%s", fmt.Sprintf("%s_%s", org, repo))))
		},
			Entry("is kubevirt/kubevirt", "kubevirt", "kubevirt"),
			Entry("is kubevirt/containerized-data-importer", "kubevirt", "containerized-data-importer"),
			Entry("is test/blah", "test", "blah"),
		)

		DescribeTable("GitHub link contains repo and org", func(org, repo string) {
			parameters := flakefinder.Params{Data: map[string]map[string]*flakefinder.Details{
				"t1": {jobNameA: &flakefinder.Details{Failed: 4, Succeeded: 1, Skipped: 2, Severity: "red", Jobs: []*flakefinder.Job{
					{BuildNumber: 1742, Severity: "red", PR: 1427, Job: "testblah"},
				}}},
			}, Headers: []string{jobNameA, "b", "c"}, Tests: []string{"t1", "t2", testName3}, EndOfReport: "2019-08-23", Org: org, Repo: repo}

			prepareBuffer(parameters)

			Expect(buffer.String()).To(ContainSubstring(fmt.Sprintf("https://github.com/%s/%s", org, repo)))
		},
			Entry("is kubevirt/kubevirt", "kubevirt", "kubevirt"),
			Entry("is kubevirt/containerized-data-importer", "kubevirt", "containerized-data-importer"),
			Entry("is test/blah", "test", "blah"),
		)

		It("shows job header table", func() {
			parameters := flakefinder.Params{Data: map[string]map[string]*flakefinder.Details{
				"t1": {jobNameA: &flakefinder.Details{Failed: 4, Succeeded: 1, Skipped: 2, Severity: "red", Jobs: []*flakefinder.Job{
					{BuildNumber: 1742, Severity: "red", PR: 1427, Job: "testblah"},
				}}},
			}, Headers: []string{jobNameA, "b", "c"}, Tests: []string{"t1", "t2", testName3}, EndOfReport: "2019-08-23", Org: "kubevirt", Repo: "kubevirt",
				FailuresForJobs: map[string]*flakefinder.JobFailures{
					"1742": {
						BuildNumber: 1742,
						PR:          17,
						Job:         "k8s-1.18-whatever",
						Failures:    66,
					},
					"4217": {
						BuildNumber: 4217,
						PR:          42,
						Job:         "k8s-1.19-whocares",
						Failures:    66,
					},
				},
			}

			prepareBuffer(parameters)

			Expect(buffer.String()).To(ContainSubstring("4217"))
			Expect(buffer.String()).To(ContainSubstring("k8s-1.18-whatever"))
			Expect(buffer.String()).To(ContainSubstring("k8s-1.19-whocares"))
		})

		It("shows batch job PRs", func() {
			parameters := flakefinder.Params{Data: map[string]map[string]*flakefinder.Details{
				"t1": {jobNameA: &flakefinder.Details{Failed: 4, Succeeded: 1, Skipped: 2, Severity: "red", Jobs: []*flakefinder.Job{
					{BuildNumber: 1742, Severity: "red", BatchPRs: []int{1427, 1737}, Job: "testblah"},
				}}},
			}, Headers: []string{jobNameA, "b", "c"}, Tests: []string{"t1", "t2", testName3}, EndOfReport: "2019-08-23", Org: "kubevirt", Repo: "kubevirt",
				FailuresForJobs: map[string]*flakefinder.JobFailures{
					"1742": {
						BuildNumber: 1742,
						BatchPRs:    []int{1427, 1737},
						Job:         "k8s-1.18-whatever",
						Failures:    66,
					},
					"4217": {
						BuildNumber: 4217,
						PR:          42,
						Job:         "k8s-1.19-whocares",
						Failures:    66,
					},
				},
			}

			prepareBuffer(parameters)

			Expect(buffer.String()).To(ContainSubstring("1427"))
			Expect(buffer.String()).To(ContainSubstring("1737"))
			Expect(buffer.String()).To(ContainSubstring("k8s-1.18-whatever"))
			Expect(buffer.String()).To(ContainSubstring("k8s-1.19-whocares"))
		})

	})

	When("rendering report csv", func() {

		var buffer bytes.Buffer

		prepareBuffer := func() {
			buffer = bytes.Buffer{}
			data := CSVParams{
				Data: map[string]map[string]*flakefinder.Details{
					"t1": {
						jobNameA: &flakefinder.Details{Failed: 4, Succeeded: 1, Skipped: 2, Severity: "red", Jobs: []*flakefinder.Job{
							{
								BuildNumber: 1742,
								Severity:    "red",
								PR:          4217,
								Job:         "testJob",
							}}},
						"b": &flakefinder.Details{Failed: 5, Succeeded: 2, Skipped: 3, Severity: "yellow", Jobs: []*flakefinder.Job{}},
					},
					"t2": {
						jobNameA: &flakefinder.Details{Failed: 8, Succeeded: 2, Skipped: 4, Severity: "cyan", Jobs: []*flakefinder.Job{}},
						"b":      &flakefinder.Details{Failed: 9, Succeeded: 3, Skipped: 5, Severity: "blue", Jobs: []*flakefinder.Job{}},
					},
				},
			}
			err := flakefinder.WriteTemplateToOutput(ReportCSVTemplate, data, &buffer)
			Expect(err).ToNot(HaveOccurred())
			if testOptions.printTestOutput {
				logger := log.New(os.Stdout, "reportCSV:", log.Flags())
				logger.Println(buffer.String())
			}
		}

		It("contains headers", func() {
			prepareBuffer()
			Expect(buffer.String()).To(ContainSubstring("\"Test Name\",\"Test Lane\",\"Severity\",\"Failed\",\"Succeeded\",\"Skipped\",\"Jobs (JSON)\""))
		})

		It("contains data", func() {
			prepareBuffer()
			Expect(buffer.String()).To(ContainSubstring("\"t1\",\"a\",\"red\",4,1,2"))
		})

		It("is valid CSV", func() {
			prepareBuffer()
			Expect(validation.CSVValidator{}.IsValid(buffer.Bytes())).To(BeNil())
		})

	})

})

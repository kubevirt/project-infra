package flakefinder

import (
	"fmt"
	"time"

	"github.com/joshdk/go-junit"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("report_data.go", func() {

	When("sorting test data", func() {
		tests := []string{"t1", "t2", "t3"}

		It("returns all tests", func() {
			data := map[string]map[string]*Details{
				"t3": {"a": &Details{Failed: 4, Succeeded: 1, Skipped: 2, Severity: HeavilyFlaky, Jobs: []*Job{}}},
			}

			Expect(SortTestsByRelevance(data, tests)).To(BeEquivalentTo([]string{"t3", "t1", "t2"}))
		})

		It("returns no duplicated tests", func() {
			data := map[string]map[string]*Details{
				"t1": {
					"a": &Details{Failed: 3, Succeeded: 1, Skipped: 2, Severity: MostlyFlaky, Jobs: []*Job{}},
					"b": &Details{Failed: 3, Succeeded: 1, Skipped: 2, Severity: Unimportant, Jobs: []*Job{}},
				},
				"t2": {"a": &Details{Failed: 2, Succeeded: 1, Skipped: 2, Severity: MildlyFlaky, Jobs: []*Job{}}},
				"t3": {"a": &Details{Failed: 4, Succeeded: 1, Skipped: 2, Severity: HeavilyFlaky, Jobs: []*Job{}}},
			}

			Expect(SortTestsByRelevance(data, tests)).To(BeEquivalentTo([]string{"t3", "t1", "t2"}))
		})

		It("returns no duplicated tests for the end", func() {
			data := map[string]map[string]*Details{
				"t1": {
					"a": &Details{Failed: 3, Succeeded: 1, Skipped: 2, Severity: MostlyFlaky, Jobs: []*Job{}},
					"b": &Details{Failed: 3, Succeeded: 1, Skipped: 2, Severity: Unimportant, Jobs: []*Job{}},
				},
				"t2": {
					"a": &Details{Failed: 2, Succeeded: 1, Skipped: 2, Severity: MildlyFlaky, Jobs: []*Job{}},
					"b": &Details{Failed: 2, Succeeded: 1, Skipped: 2, Severity: Unimportant, Jobs: []*Job{}},
				},
				"t3": {"a": &Details{Failed: 4, Succeeded: 1, Skipped: 2, Severity: HeavilyFlaky, Jobs: []*Job{}}},
			}

			Expect(SortTestsByRelevance(data, tests)).To(BeEquivalentTo([]string{"t3", "t1", "t2"}))
		})

		It("returns tests sorted descending by severity", func() {
			data := map[string]map[string]*Details{
				"t1": {"a": &Details{Failed: 3, Succeeded: 1, Skipped: 2, Severity: MostlyFlaky, Jobs: []*Job{}}},
				"t2": {"a": &Details{Failed: 2, Succeeded: 1, Skipped: 2, Severity: MildlyFlaky, Jobs: []*Job{}}},
				"t3": {"a": &Details{Failed: 4, Succeeded: 1, Skipped: 2, Severity: HeavilyFlaky, Jobs: []*Job{}}},
			}

			Expect(SortTestsByRelevance(data, tests)).To(BeEquivalentTo([]string{"t3", "t1", "t2"}))
		})

		It("returns tests of same severity sorted descending by number of severity points", func() {
			data := map[string]map[string]*Details{
				"t1": {
					"a": &Details{Failed: 3, Succeeded: 1, Skipped: 2, Severity: HeavilyFlaky, Jobs: []*Job{}},
					"b": &Details{Failed: 3, Succeeded: 1, Skipped: 2, Severity: MostlyFlaky, Jobs: []*Job{}},
				},
				"t2": {
					"a": &Details{Failed: 2, Succeeded: 1, Skipped: 2, Severity: HeavilyFlaky, Jobs: []*Job{}},
					"b": &Details{Failed: 2, Succeeded: 1, Skipped: 2, Severity: MildlyFlaky, Jobs: []*Job{}},
				},
				"t3": {
					"a": &Details{Failed: 4, Succeeded: 1, Skipped: 2, Severity: HeavilyFlaky, Jobs: []*Job{}},
					"b": &Details{Failed: 4, Succeeded: 1, Skipped: 2, Severity: HeavilyFlaky, Jobs: []*Job{}},
				},
			}

			Expect(SortTestsByRelevance(data, tests)).To(BeEquivalentTo([]string{"t3", "t1", "t2"}))
		})

		type testData struct {
			testName  string
			failed    []int
			succeeded []int
			skipped   []int
			severity  string
		}

		DescribeTable("returns tests of same severity weighted by total number of tests", func(expectations []testData, expectedTestNameOrder []string) {
			testData := map[string]map[string]*Details{}
			for _, expectation := range expectations {
				testData[expectation.testName] = map[string]*Details{}
				for index, failed := range expectation.failed {
					testData[expectation.testName][fmt.Sprint(index)] = &Details{Failed: failed, Succeeded: expectation.succeeded[index], Skipped: expectation.skipped[index], Severity: expectation.severity, Jobs: []*Job{}}
				}
			}

			Expect(SortTestsByRelevance(testData, tests)).To(BeEquivalentTo(expectedTestNameOrder))
		},
			Entry("zeros shouldn't be a problem",
				[]testData{
					{testName: "t1", failed: []int{2}, succeeded: []int{0}, skipped: []int{2}, severity: HeavilyFlaky},
					{testName: "t2", failed: []int{1}, succeeded: []int{0}, skipped: []int{2}, severity: HeavilyFlaky},
					{testName: "t3", failed: []int{3}, succeeded: []int{0}, skipped: []int{2}, severity: HeavilyFlaky},
				},
				[]string{"t3", "t1", "t2"},
			),
			Entry("the more failures the higher",
				[]testData{
					{testName: "t1", failed: []int{2}, succeeded: []int{1}, skipped: []int{2}, severity: HeavilyFlaky},
					{testName: "t2", failed: []int{1}, succeeded: []int{1}, skipped: []int{2}, severity: HeavilyFlaky},
					{testName: "t3", failed: []int{3}, succeeded: []int{1}, skipped: []int{2}, severity: HeavilyFlaky},
				},
				[]string{"t3", "t1", "t2"},
			),
			Entry("multiple values with zeros",
				[]testData{
					{testName: "t1", failed: []int{2, 0}, succeeded: []int{1, 0}, skipped: []int{2, 2}, severity: HeavilyFlaky},
					{testName: "t2", failed: []int{1, 0}, succeeded: []int{1, 0}, skipped: []int{2, 2}, severity: HeavilyFlaky},
					{testName: "t3", failed: []int{3, 0}, succeeded: []int{1, 0}, skipped: []int{2, 2}, severity: HeavilyFlaky},
				},
				[]string{"t3", "t1", "t2"},
			),
			Entry("multiple values",
				[]testData{
					{testName: "t1", failed: []int{4, 5}, succeeded: []int{1, 1}, skipped: []int{2, 2}, severity: HeavilyFlaky},
					{testName: "t2", failed: []int{3, 4}, succeeded: []int{1, 1}, skipped: []int{2, 2}, severity: HeavilyFlaky},
					{testName: "t3", failed: []int{6, 7}, succeeded: []int{1, 1}, skipped: []int{2, 2}, severity: HeavilyFlaky},
				},
				[]string{"t3", "t1", "t2"},
			),
			Entry("errors high, ratios small",
				[]testData{
					{testName: "t1", failed: []int{6, 7}, succeeded: []int{5, 6}, skipped: []int{2, 2}, severity: HeavilyFlaky},
					{testName: "t2", failed: []int{4, 5}, succeeded: []int{3, 4}, skipped: []int{2, 2}, severity: HeavilyFlaky},
					{testName: "t3", failed: []int{11, 12}, succeeded: []int{10, 11}, skipped: []int{2, 2}, severity: HeavilyFlaky},
				},
				[]string{"t3", "t1", "t2"},
			),
			Entry("higher ratio, the higher",
				[]testData{
					{testName: "t1", failed: []int{6, 7}, succeeded: []int{3, 4}, skipped: []int{2, 2}, severity: HeavilyFlaky},
					{testName: "t2", failed: []int{4, 5}, succeeded: []int{2, 3}, skipped: []int{2, 2}, severity: HeavilyFlaky},
					{testName: "t3", failed: []int{11, 12}, succeeded: []int{2, 1}, skipped: []int{2, 2}, severity: HeavilyFlaky},
				},
				[]string{"t3", "t1", "t2"},
			),
			Entry("mixed lengths",
				[]testData{
					{testName: "t1", failed: []int{8, 10}, succeeded: []int{2, 1}, skipped: []int{2, 2}, severity: HeavilyFlaky},
					{testName: "t2", failed: []int{3, 4, 5}, succeeded: []int{1, 2, 3}, skipped: []int{2, 2, 2}, severity: HeavilyFlaky},
					{testName: "t3", failed: []int{22}, succeeded: []int{2}, skipped: []int{2}, severity: HeavilyFlaky},
				},
				[]string{"t3", "t1", "t2"},
			),

			// while the ratio here is higher for t2, we still want to emphazise the cases with higher failures more
			Entry("real life case 1: 10/9/2 > 9/9/3 > 6/3/12",
				[]testData{
					{testName: "t1", failed: []int{9}, succeeded: []int{9}, skipped: []int{3}, severity: HeavilyFlaky},
					{testName: "t2", failed: []int{6}, succeeded: []int{3}, skipped: []int{12}, severity: HeavilyFlaky},
					{testName: "t3", failed: []int{10}, succeeded: []int{9}, skipped: []int{2}, severity: HeavilyFlaky},
				},
				[]string{"t3", "t1", "t2"},
			),
			Entry("real life case 2: 6/3/12 > 5/0/2 > 4/1/2",
				[]testData{
					{testName: "t1", failed: []int{5}, succeeded: []int{0}, skipped: []int{2}, severity: HeavilyFlaky},
					{testName: "t2", failed: []int{4}, succeeded: []int{1}, skipped: []int{2}, severity: HeavilyFlaky},
					{testName: "t3", failed: []int{6}, succeeded: []int{3}, skipped: []int{12}, severity: HeavilyFlaky},
				},
				[]string{"t3", "t1", "t2"},
			),
			Entry("real life case 1: 15/1/0 > 13/0/3 > 3/0/13 > ",
				[]testData{
					{testName: "t1", failed: []int{13}, succeeded: []int{0}, skipped: []int{3}, severity: HeavilyFlaky},
					{testName: "t2", failed: []int{3}, succeeded: []int{0}, skipped: []int{13}, severity: HeavilyFlaky},
					{testName: "t3", failed: []int{15}, succeeded: []int{1}, skipped: []int{0}, severity: HeavilyFlaky},
				},
				[]string{"t3", "t1", "t2"},
			),

			Entry("less number of succeeded, more severe: 5/1/13 > 5/3/3 > 5/5/0",
				[]testData{
					{testName: "t1", failed: []int{5}, succeeded: []int{3}, skipped: []int{3}, severity: HeavilyFlaky},
					{testName: "t2", failed: []int{5}, succeeded: []int{5}, skipped: []int{13}, severity: HeavilyFlaky},
					{testName: "t3", failed: []int{5}, succeeded: []int{1}, skipped: []int{0}, severity: HeavilyFlaky},
				},
				[]string{"t3", "t1", "t2"},
			),
		)

	})

	When("sorting test via severity", func() {

		It("returns tests of same severity sorted descending by number of severity points", func() {
			Expect(BuildUpSortedTestsBySeverity(map[string]map[string]int{
				"t1": {HeavilyFlaky: 2},
				"t2": {HeavilyFlaky: 1},
				"t3": {HeavilyFlaky: 3},
			})).To(BeEquivalentTo([]string{"t3", "t1", "t2"}))
		})

		It("returns tests of same severity and same number of severity points sorted lexically", func() {
			Expect(BuildUpSortedTestsBySeverity(map[string]map[string]int{
				"tb": {HeavilyFlaky: 2},
				"tc": {HeavilyFlaky: 2},
				"ta": {HeavilyFlaky: 2},
			})).To(BeEquivalentTo([]string{"ta", "tb", "tc"}))
		})

		It("returns tests of same severity sorted by lower severity", func() {
			Expect(BuildUpSortedTestsBySeverity(map[string]map[string]int{
				"tb": {HeavilyFlaky: 2, MostlyFlaky: 2},
				"tc": {HeavilyFlaky: 2, MostlyFlaky: 1},
				"ta": {HeavilyFlaky: 2, MostlyFlaky: 3},
			})).To(BeEquivalentTo([]string{"ta", "tb", "tc"}))
		})

		It("returns tests of same severity sorted by lower severity if even lower values present but zero", func() {
			Expect(BuildUpSortedTestsBySeverity(map[string]map[string]int{
				"tb": {HeavilyFlaky: 2, MostlyFlaky: 2, ModeratelyFlaky: 0, MildlyFlaky: 0},
				"tc": {HeavilyFlaky: 2, MostlyFlaky: 1, ModeratelyFlaky: 0, MildlyFlaky: 0},
				"ta": {HeavilyFlaky: 2, MostlyFlaky: 3, ModeratelyFlaky: 0, MildlyFlaky: 0},
			})).To(BeEquivalentTo([]string{"ta", "tb", "tc"}))
		})

		It("returns tests of same severity sorted by lower severity if inbetween values present but zero", func() {
			Expect(BuildUpSortedTestsBySeverity(map[string]map[string]int{
				"tb": {HeavilyFlaky: 2, MostlyFlaky: 0, ModeratelyFlaky: 2, MildlyFlaky: 0},
				"tc": {HeavilyFlaky: 2, MostlyFlaky: 0, ModeratelyFlaky: 1, MildlyFlaky: 0},
				"ta": {HeavilyFlaky: 2, MostlyFlaky: 0, ModeratelyFlaky: 3, MildlyFlaky: 0},
			})).To(BeEquivalentTo([]string{"ta", "tb", "tc"}))
		})

		It("returns tests of same severity sorted by lower severity if some inbetween values zero", func() {
			Expect(BuildUpSortedTestsBySeverity(map[string]map[string]int{
				"tb": {HeavilyFlaky: 2, MostlyFlaky: 0, ModeratelyFlaky: 2, MildlyFlaky: 0},
				"tc": {HeavilyFlaky: 2, MostlyFlaky: 0, ModeratelyFlaky: 0, MildlyFlaky: 0},
				"ta": {HeavilyFlaky: 2, MostlyFlaky: 3, ModeratelyFlaky: 0, MildlyFlaky: 0},
			})).To(BeEquivalentTo([]string{"ta", "tb", "tc"}))
		})

		It("returns tests of same severity sorted by lower severity if some inbetween values zero with more values", func() {
			Expect(BuildUpSortedTestsBySeverity(map[string]map[string]int{
				"tb": {HeavilyFlaky: 1, MostlyFlaky: 0, ModeratelyFlaky: 0, MildlyFlaky: 1, Fine: 0, Unimportant: 0},
				"tc": {HeavilyFlaky: 1, MostlyFlaky: 0, ModeratelyFlaky: 0, MildlyFlaky: 0, Fine: 0, Unimportant: 0},
				"ta": {HeavilyFlaky: 1, MostlyFlaky: 0, ModeratelyFlaky: 0, MildlyFlaky: 2, Fine: 0, Unimportant: 0},
			})).To(BeEquivalentTo([]string{"ta", "tb", "tc"}))
		})

	})

	DescribeTable("When comparing severity",
		func(a, b *TestToSeverityOccurrences, expected bool) {
			bySeverity := []*TestToSeverityOccurrences{a, b}
			Expect(BySeverity.Less(bySeverity, 0, 1)).To(BeEquivalentTo(expected))
		},

		Entry("ta -> Sev(2) less than tb -> Sev(2) is false",
			&TestToSeverityOccurrences{Name: "ta", SeverityOccurrences: []int{2}},
			&TestToSeverityOccurrences{Name: "tb", SeverityOccurrences: []int{2}},
			false,
		),
		Entry("tb -> Sev(2) less than ta -> Sev(2) is true",
			&TestToSeverityOccurrences{Name: "tb", SeverityOccurrences: []int{2}},
			&TestToSeverityOccurrences{Name: "ta", SeverityOccurrences: []int{2}},
			true,
		),
		Entry("ta -> Sev(2) less than ta -> Sev(2) is false",
			&TestToSeverityOccurrences{Name: "ta", SeverityOccurrences: []int{2}},
			&TestToSeverityOccurrences{Name: "ta", SeverityOccurrences: []int{2}},
			false,
		),
		Entry("tb -> Sev(3) is less than ta -> Sev(2) is false",
			&TestToSeverityOccurrences{Name: "tb", SeverityOccurrences: []int{3}},
			&TestToSeverityOccurrences{Name: "ta", SeverityOccurrences: []int{2}},
			false,
		),
		Entry("ta -> Sev(2) is less than tb -> Sev(3) is true",
			&TestToSeverityOccurrences{Name: "ta", SeverityOccurrences: []int{2}},
			&TestToSeverityOccurrences{Name: "tb", SeverityOccurrences: []int{3}},
			true,
		),
		Entry("tb -> Sev(3, 2) is less ta -> Sev(3, 3) is true",
			&TestToSeverityOccurrences{Name: "tb", SeverityOccurrences: []int{3, 2}},
			&TestToSeverityOccurrences{Name: "ta", SeverityOccurrences: []int{3, 3}},
			true,
		),
		Entry("tb -> Sev(3, 3) is less ta -> Sev(3, 2) is false",
			&TestToSeverityOccurrences{Name: "tb", SeverityOccurrences: []int{3, 3}},
			&TestToSeverityOccurrences{Name: "ta", SeverityOccurrences: []int{3, 2}},
			false,
		),
		Entry("tb -> Sev(3, 0, 3) is less ta -> Sev(3, 0, 2) is false",
			&TestToSeverityOccurrences{Name: "tb", SeverityOccurrences: []int{3, 0, 3}},
			&TestToSeverityOccurrences{Name: "ta", SeverityOccurrences: []int{3, 0, 2}},
			false,
		),
		Entry("tb -> Sev(3, 0, 2) is not less ta -> Sev(3, 0, 3) is true",
			&TestToSeverityOccurrences{Name: "tb", SeverityOccurrences: []int{3, 0, 2}},
			&TestToSeverityOccurrences{Name: "ta", SeverityOccurrences: []int{3, 0, 3}},
			true,
		),
		Entry("tb -> Sev(1,0,0,2,0,0) is less ta -> Sev(1,0,0,1,0,0) is false",
			&TestToSeverityOccurrences{Name: "tb", SeverityOccurrences: []int{1, 0, 0, 2, 0, 0}},
			&TestToSeverityOccurrences{Name: "ta", SeverityOccurrences: []int{1, 0, 0, 1, 0, 0}},
			false,
		),
		Entry("ta -> Sev(1,0,0,1,0,0) is less tb -> Sev(1,0,0,2,0,0) is true",
			&TestToSeverityOccurrences{Name: "ta", SeverityOccurrences: []int{1, 0, 0, 1, 0, 0}},
			&TestToSeverityOccurrences{Name: "tb", SeverityOccurrences: []int{1, 0, 0, 2, 0, 0}},
			true,
		),
		Entry("tb -> Sev(1,0,1,0,0,0) is less ta -> Sev(1,0,0,1,0,0) is false",
			&TestToSeverityOccurrences{Name: "tb", SeverityOccurrences: []int{1, 0, 1, 0, 0, 0}},
			&TestToSeverityOccurrences{Name: "ta", SeverityOccurrences: []int{1, 0, 0, 1, 0, 0}},
			false,
		),
		Entry("ta -> Sev(1,0,0,1,0,0) is less tb -> Sev(1,0,1,0,0,0) is true",
			&TestToSeverityOccurrences{Name: "ta", SeverityOccurrences: []int{1, 0, 0, 1, 0, 0}},
			&TestToSeverityOccurrences{Name: "tb", SeverityOccurrences: []int{1, 0, 1, 0, 0, 0}},
			true,
		),
	)

	When("swapping elements", func() {

		It("Works", func() {
			bySeverity := []*TestToSeverityOccurrences{
				{Name: "tb", SeverityOccurrences: []int{3, 0, 2}},
				{Name: "ta", SeverityOccurrences: []int{3, 0, 3}},
			}
			BySeverity.Swap(bySeverity, 0, 1)
			Expect(bySeverity[0].Name).To(BeEquivalentTo("ta"))
			Expect(bySeverity[1].Name).To(BeEquivalentTo("tb"))
		})

	})

	DescribeTable("When calculating severity",
		func(details *Details, expected string) {
			SetSeverity(details)
			Expect(details.Severity).To(BeEquivalentTo(expected))
		},
		Entry("results having no failed tests but successful tests is fine", &Details{Failed: 0, Succeeded: 1, Skipped: 2, Jobs: []*Job{}}, Fine),
		Entry("results having no successful tests is heavily flaky", &Details{Failed: 1, Succeeded: 0, Skipped: 2, Jobs: []*Job{}}, HeavilyFlaky),
		Entry("results being HeavilyFlaky", &Details{Failed: 1, Succeeded: 1, Skipped: 2, Jobs: []*Job{}}, HeavilyFlaky),
		Entry("results being MostlyFlaky", &Details{Failed: 1, Succeeded: 2, Skipped: 2, Jobs: []*Job{}}, MostlyFlaky),
		Entry("results being ModeratelyFlaky", &Details{Failed: 1, Succeeded: 5, Skipped: 2, Jobs: []*Job{}}, ModeratelyFlaky),
		Entry("results being MildlyFlaky", &Details{Failed: 1, Succeeded: 10, Skipped: 2, Jobs: []*Job{}}, MildlyFlaky),
	)

	When("massaging test data", func() {

		const org = "org"
		const repo = "repo"
		const pr = 17
		const buildNumber = 1742

		var minusDay time.Duration

		BeforeEach(func() {
			duration, err := time.ParseDuration("24h")
			if err != nil {
				panic(err)
			}
			minusDay = -1 * duration
		})

		It("creates a result for PRs", func() {
			startOfReport := time.Now().Add(minusDay)
			endOfReport := time.Now()
			Expect(CreateFlakeReportData(
				[]*JobResult{
					{
						Job: "job",
						JUnit: []junit.Suite{
							{
								Name:       "suite",
								Package:    "",
								Properties: nil,
								Tests: []junit.Test{
									{
										Name:       "test1",
										Classname:  "",
										Duration:   0,
										Status:     junit.StatusPassed,
										Error:      nil,
										Properties: nil,
									},
									{
										Name:       "test2",
										Classname:  "",
										Duration:   0,
										Status:     junit.StatusSkipped,
										Error:      nil,
										Properties: nil,
									},
									{
										Name:       "test3",
										Classname:  "",
										Duration:   0,
										Status:     junit.StatusFailed,
										Error:      nil,
										Properties: nil,
									},
								},
								SystemOut: "",
								SystemErr: "",
								Totals:    junit.Totals{},
							},
						},
						BuildNumber: buildNumber,
						PR:          pr,
						BatchPRs:    nil,
					},
				},
				[]int{pr},
				endOfReport,
				org,
				repo,
				startOfReport,
			)).To(BeEquivalentTo(
				Params{
					StartOfReport: startOfReport.Format(time.RFC3339),
					EndOfReport:   endOfReport.Format(time.RFC3339),
					Headers:       []string{"job"},
					Tests:         []string{"test3"},
					TestAttributes: map[string]TestAttributes{
						"test3": nil,
					},
					Data: map[string]map[string]*Details{
						"test3": {
							"job": {
								Succeeded: 0,
								Skipped:   0,
								Failed:    1,
								Severity:  "red",
								Jobs: []*Job{
									{BuildNumber: buildNumber, Severity: "red", PR: pr, BatchPRs: nil, Job: "job"},
								},
							},
						},
					},
					PrNumbers: []int{pr},
					Org:       org,
					Repo:      repo,
					FailuresForJobs: map[string]*JobFailures{
						fmt.Sprintf("job-%d", buildNumber): {BuildNumber: buildNumber, PR: pr, BatchPRs: nil, Job: "job", Failures: 1},
					},
					BareTestNames: map[string]string{"test3": "test3"},
				}))
		})

		It("creates a result for batch PRs", func() {
			startOfReport := time.Now().Add(minusDay)
			endOfReport := time.Now()
			Expect(CreateFlakeReportData(
				[]*JobResult{
					{
						Job: "job",
						JUnit: []junit.Suite{
							{
								Name:       "suite",
								Package:    "",
								Properties: nil,
								Tests: []junit.Test{
									{
										Name:       "test1",
										Classname:  "",
										Duration:   0,
										Status:     junit.StatusPassed,
										Error:      nil,
										Properties: nil,
									},
									{
										Name:       "test2",
										Classname:  "",
										Duration:   0,
										Status:     junit.StatusSkipped,
										Error:      nil,
										Properties: nil,
									},
									{
										Name:       "test3",
										Classname:  "",
										Duration:   0,
										Status:     junit.StatusFailed,
										Error:      nil,
										Properties: nil,
									},
								},
								SystemOut: "",
								SystemErr: "",
								Totals:    junit.Totals{},
							},
						},
						BuildNumber: buildNumber,
						PR:          0,
						BatchPRs:    []int{pr},
					},
				},
				[]int{pr},
				endOfReport,
				org,
				repo,
				startOfReport,
			)).To(BeEquivalentTo(
				Params{
					StartOfReport: startOfReport.Format(time.RFC3339),
					EndOfReport:   endOfReport.Format(time.RFC3339),
					Headers:       []string{"job"},
					Tests:         []string{"test3"},
					TestAttributes: map[string]TestAttributes{
						"test3": nil,
					},
					Data: map[string]map[string]*Details{
						"test3": {
							"job": {
								Succeeded: 0,
								Skipped:   0,
								Failed:    1,
								Severity:  "red",
								Jobs: []*Job{
									{BuildNumber: buildNumber, Severity: "red", PR: 0, BatchPRs: []int{pr}, Job: "job"},
								},
							},
						},
					},
					PrNumbers: []int{pr},
					Org:       org,
					Repo:      repo,
					FailuresForJobs: map[string]*JobFailures{
						fmt.Sprintf("job-%d", buildNumber): {BuildNumber: buildNumber, PR: 0, BatchPRs: []int{pr}, Job: "job", Failures: 1},
					},
					BareTestNames: map[string]string{"test3": "test3"},
				}))
		})

		It("adds test attributes for failing tests", func() {
			startOfReport := time.Now().Add(minusDay)
			endOfReport := time.Now()
			Expect(CreateFlakeReportData(
				[]*JobResult{
					{
						Job: "job",
						JUnit: []junit.Suite{
							{
								Name:       "suite",
								Package:    "",
								Properties: nil,
								Tests: []junit.Test{
									{
										Name:       "test1",
										Classname:  "",
										Duration:   0,
										Status:     junit.StatusPassed,
										Error:      nil,
										Properties: nil,
									},
									{
										Name:       "test2",
										Classname:  "",
										Duration:   0,
										Status:     junit.StatusSkipped,
										Error:      nil,
										Properties: nil,
									},
									{
										Name:       "[Serial]test3[sig-compute]",
										Classname:  "",
										Duration:   0,
										Status:     junit.StatusFailed,
										Error:      nil,
										Properties: nil,
									},
								},
								SystemOut: "",
								SystemErr: "",
								Totals:    junit.Totals{},
							},
						},
						BuildNumber: buildNumber,
						PR:          pr,
						BatchPRs:    nil,
					},
				},
				[]int{pr},
				endOfReport,
				org,
				repo,
				startOfReport,
			)).To(BeEquivalentTo(
				Params{
					StartOfReport: startOfReport.Format(time.RFC3339),
					EndOfReport:   endOfReport.Format(time.RFC3339),
					Headers:       []string{"job"},
					Tests:         []string{"[Serial]test3[sig-compute]"},
					TestAttributes: map[string]TestAttributes{
						"[Serial]test3[sig-compute]": {
							{Name: "Serial", Value: "", AttributeType: 2},
							{Name: "sig-compute", Value: "", AttributeType: 3},
						},
					},
					Data: map[string]map[string]*Details{
						"[Serial]test3[sig-compute]": {
							"job": {
								Succeeded: 0,
								Skipped:   0,
								Failed:    1,
								Severity:  "red",
								Jobs: []*Job{
									{BuildNumber: buildNumber, Severity: "red", PR: pr, BatchPRs: nil, Job: "job"},
								},
							},
						},
					},
					PrNumbers: []int{pr},
					Org:       org,
					Repo:      repo,
					FailuresForJobs: map[string]*JobFailures{
						fmt.Sprintf("job-%d", buildNumber): {BuildNumber: buildNumber, PR: pr, BatchPRs: nil, Job: "job", Failures: 1},
					},
					BareTestNames: map[string]string{
						"[Serial]test3[sig-compute]": "test3",
					},
				}))
		})

	})

	DescribeTable("Extracting TestAttributes", func(testName string, expected TestAttributes) {
		Expect(NewTestAttributes(testName)).To(BeEquivalentTo(expected))
	},
		Entry("empty", "", nil),
		Entry("serial", "[Serial]", TestAttributes{
			{
				Name:          "Serial",
				Value:         "",
				AttributeType: TestAttributeTypeSerial,
			},
		}),
		Entry("sig", "[sig-storage]", TestAttributes{
			{
				Name:          "sig-storage",
				Value:         "",
				AttributeType: TestAttributeTypeSIG,
			},
		}),
		Entry("test id", "[test_id:1742]", TestAttributes{
			{
				Name:          "test_id",
				Value:         "1742",
				AttributeType: TestAttributeTypeTestID,
			},
		}),
		Entry("quarantine", "[QUARANTINE]", TestAttributes{
			{
				Name:          "QUARANTINE",
				Value:         "",
				AttributeType: TestAttributeTypeQuarantine,
			},
		}),
		Entry("release-blocker", "[release-blocker]", TestAttributes{
			{
				Name:          "release-blocker",
				Value:         "",
				AttributeType: TestAttributeTypeReleaseBlocker,
			},
		}),
		Entry("complex test name", "[Serial][sig-operator]Operator [rfe_id:2291][crit:high][vendor:cnv-qe@redhat.com][level:component]should update kubevirt [release-blocker][test_id:3145]from previous release to target tested release by patching KubeVirt CR", TestAttributes{
			{
				Name:          "release-blocker",
				Value:         "",
				AttributeType: 0,
			},
			{Name: "Serial", Value: "", AttributeType: 2},
			{Name: "sig-operator", Value: "", AttributeType: 3},
			{Name: "test_id", Value: "3145", AttributeType: 4},
			{Name: "rfe_id", Value: "2291", AttributeType: 5},
			{Name: "crit", Value: "high", AttributeType: 5},
			{
				Name:          "vendor",
				Value:         "cnv-qe@redhat.com",
				AttributeType: 5,
			},
			{Name: "level", Value: "component", AttributeType: 5},
		}),
	)

	DescribeTable("Getting bare test names", func(testName string, expected string) {
		Expect(GetBareTestName(testName)).To(BeEquivalentTo(expected))
	},
		Entry("empty", "", ""),
		Entry("complex test name", "[Serial][sig-operator]Operator [rfe_id:2291][crit:high][vendor:cnv-qe@redhat.com][level:component]should update kubevirt [release-blocker][test_id:3145]from previous release to target tested release by patching KubeVirt CR", "Operator should update kubevirt from previous release to target tested release by patching KubeVirt CR"),
	)

})

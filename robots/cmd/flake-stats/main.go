package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"kubevirt.io/project-infra/robots/pkg/flakefinder"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

//go:embed flake-stats.gohtml
var htmlTemplate string

var multipleSpacesRegex = regexp.MustCompile(`\s+`)

type TemplateData struct {
	OverallFailures *TopXTest
	TopXTests
	DaysInThePast   int
	Date            time.Time
	ShareCategories []ShareCategory
	Org             string
	Repo            string
}

type TopXTests []*TopXTest

type ShareCategory struct {
	CSSClassName       string
	MinPercentageValue float64
}

var shareCategories = []ShareCategory{
	{
		"lightyellow",
		0.25,
	},
	{
		"yellow",
		1.0,
	},
	{
		"orange",
		2.5,
	},
	{
		"orangered",
		5.0,
	},
	{
		"red",
		10.0,
	},
	{
		"darkred",
		25.0,
	},
}

func (t TopXTests) Len() int {
	return len(t)
}

func (t TopXTests) Less(i, j int) bool {

	// go through the FailuresPerDay from most recent to last
	// the one which has more recent failures is less than the other
	// where "more recent failures" means per i,j that
	// 1) mostRecentSetOfFailures := most recent complete set
	//    of directly adjacent failures per each day
	//    i.e. assuming today is Wed
	//         thus [Wed, Tue, Mon] is more recent than [Tue, Mon, Sun]
	// 2) sum(mostRecentSetOfFailures(i)) > sum(mostRecentSetOfFailures(j))
	tIFailuresPerDaySum, tJFailuresPerDaySum := 0, 0
	for day := 0; day < opts.daysInThePast; day++ {
		dayForFailure := time.Now().Add(time.Duration(-1*day*24) * time.Hour)
		dateKeyForFailure := dayForFailure.Format(rfc3339Date) + "T00:00:00Z"
		tIFailuresPerDay, iExists := t[i].FailuresPerDay[dateKeyForFailure]
		tJFailuresPerDay, jExists := t[j].FailuresPerDay[dateKeyForFailure]
		if !iExists && !jExists {
			if tIFailuresPerDaySum > tJFailuresPerDaySum {
				return true
			}
			continue
		}
		if !jExists {
			return true
		}
		if !iExists {
			return false
		}
		tIFailuresPerDaySum += tIFailuresPerDay.Sum
		tJFailuresPerDaySum += tJFailuresPerDay.Sum
	}

	// continue comparing the remaining values
	iAllFailures := t[i].AllFailures
	jAllFailures := t[j].AllFailures
	return iAllFailures.Sum > jAllFailures.Sum ||
		(iAllFailures.Sum == jAllFailures.Sum && iAllFailures.Max > jAllFailures.Max) ||
		(iAllFailures.Sum == jAllFailures.Sum && iAllFailures.Max == jAllFailures.Max && iAllFailures.Avg > jAllFailures.Avg)
}

func (t TopXTests) calculateWeightedDatedFailureSums(i int, firstDayOfReport time.Time) int {
	tiFailuresPerDay := t[i].FailuresPerDay
	iWeightedDatedFailureSums := 0
	for iDate, iFailuresPerDay := range tiFailuresPerDay {
		parse, err := time.Parse(time.RFC3339, iDate)
		if err != nil {
			panic(err)
		}
		daysAfterStart := int(parse.Sub(firstDayOfReport).Hours()) / 24
		iWeightedDatedFailureSums += daysAfterStart * daysAfterStart * iFailuresPerDay.Sum
	}
	return iWeightedDatedFailureSums
}

func (t TopXTests) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t TopXTests) CalculateShareFromTotalFailures() *TopXTest {
	overall := &TopXTest{
		Name:            "Test failures overall",
		AllFailures:     &FailureCounter{Name: "overall"},
		FailuresPerDay:  map[string]*FailureCounter{},
		FailuresPerLane: map[string]*FailureCounter{},
	}
	for _, test := range t {
		overall.AllFailures.add(test.AllFailures.Sum)

		// aggregate failures per test per day
		for day, failuresPerDay := range test.FailuresPerDay {
			_, failuresPerDayExists := overall.FailuresPerDay[day]
			if !failuresPerDayExists {
				overall.FailuresPerDay[day] = &FailureCounter{
					Name: failuresPerDay.Name,
					URL: fmt.Sprintf(
						"https://storage.googleapis.com/kubevirt-prow/reports/flakefinder/kubevirt/kubevirt/flakefinder-%s-024h.html",
						formatFromRFC3339ToRFCDate(day),
					),
				}
			}
			overall.FailuresPerDay[day].add(failuresPerDay.Sum)
		}

		// aggregate failures per test per lane
		for lane, failuresPerLane := range test.FailuresPerLane {
			_, failuresPerLaneExists := overall.FailuresPerLane[lane]
			if !failuresPerLaneExists {
				overall.FailuresPerLane[lane] = &FailureCounter{
					Name: lane,
					URL:  generateTestGridURLForJob(lane),
				}
			}
			overall.FailuresPerLane[lane].add(failuresPerLane.Sum)
		}

	}
	for index := range t {
		t[index].CalculateShareFromTotalFailures(overall.AllFailures.Sum)
	}
	overall.CalculateShareFromTotalFailures(overall.AllFailures.Sum)
	return overall
}

type TopXTest struct {
	Name                   string
	AllFailures            *FailureCounter
	FailuresPerDay         map[string]*FailureCounter
	FailuresPerLane        map[string]*FailureCounter
	NoteHasBeenQuarantined bool
}

func (t *TopXTest) CalculateShareFromTotalFailures(totalFailures int) {
	t.AllFailures.setShare(totalFailures)
	for key := range t.FailuresPerLane {
		t.FailuresPerLane[key].setShare(totalFailures)
	}
	for key := range t.FailuresPerDay {
		t.FailuresPerDay[key].setShare(totalFailures)
	}
}

type FailureCounter struct {
	Name          string
	Count         int
	Sum           int
	Avg           float64
	Max           int
	SharePercent  float64
	ShareCategory ShareCategory
	URL           string
}

func (c *FailureCounter) add(value int) {
	c.Sum += value
	if value > c.Max {
		c.Max = value
	}
	c.Avg = (float64(value) + float64(c.Count)*c.Avg) / float64(c.Count+1)
	c.Count++
}

func (f *FailureCounter) setShare(totalFailures int) {
	f.SharePercent = float64(f.Sum) / float64(totalFailures) * 100
	for _, shareCategory := range shareCategories {
		if shareCategory.MinPercentageValue <= f.SharePercent {
			f.ShareCategory = shareCategory
		}
	}
}

type options struct {
	daysInThePast       int
	outputFile          string
	overwriteOutputFile bool
	org                 string
	repo                string
}

func (o options) validate() error {
	if opts.daysInThePast <= 0 {
		return fmt.Errorf("invalid value for daysInThePast %d", opts.daysInThePast)
	}
	if opts.outputFile == "" {
		file, err := os.CreateTemp("", "flake-stats-*.html")
		if err != nil {
			return fmt.Errorf("failed to generate temp file: %v", err)
		}
		opts.outputFile = file.Name()
	} else {
		if !opts.overwriteOutputFile {
			stats, err := os.Stat(opts.outputFile)
			if stats != nil || !os.IsNotExist(err) {
				return fmt.Errorf("file %q exists or error occurred: %v", opts.outputFile, err)
			}
		}
	}
	return nil
}

var opts = options{}

const defaultDaysInThePast = 14
const defaultOrg = "kubevirt"
const defaultRepo = "kubevirt"
const dayFormat = "Mon, 02 Jan 2006"
const rfc3339Date = "2006-01-02"

func main() {

	flag.IntVar(&opts.daysInThePast, "days-in-the-past", defaultDaysInThePast, "determines how much days in the past till today are covered")
	flag.StringVar(&opts.outputFile, "output-file", "", "outputfile to write to, default is a tempfile in folder")
	flag.BoolVar(&opts.overwriteOutputFile, "overwrite-output-file", false, "whether outputfile is set to be overwritten if it exists")
	flag.StringVar(&opts.org, "org", defaultOrg, "GitHub org to use for fetching report data from gcs dir")
	flag.StringVar(&opts.repo, "repo", defaultRepo, "GitHub repo to use for fetching report data from gcs dir")
	flag.Parse()

	err := opts.validate()
	if err != nil {
		log.Fatalf("failed to validate flags: %v", err)
	}

	recentFlakeFinderReports := fetchFlakeFinder24hReportsForRecentDays()
	allTests := aggregateTopXTests(recentFlakeFinderReports)
	overallFailures := allTests.CalculateShareFromTotalFailures()
	err = writeReport(overallFailures, allTests)
	if err != nil {
		log.Fatalf("failed writing report: %v", err)
	}

}

func fetchFlakeFinder24hReportsForRecentDays() []*flakefinder.Params {
	var recentFlakeFinderReports []*flakefinder.Params
	targetReportDate := previousDay(time.Now())
	for i := 0; i < opts.daysInThePast; i++ {
		flakeFinderReportData, err := fetchFlakeFinder24hReportData(targetReportDate)
		if err != nil {
			log.Fatalf("failed to retrieve flakefinder report data for %v: %v", targetReportDate, err)
		}
		recentFlakeFinderReports = append(recentFlakeFinderReports, flakeFinderReportData)

		targetReportDate = previousDay(targetReportDate)
	}
	return recentFlakeFinderReports
}

func fetchFlakeFinder24hReportData(targetReportDate time.Time) (*flakefinder.Params, error) {
	reportJSONURL, err := flakefinder.GenerateReportURL(opts.org, opts.repo, targetReportDate, flakefinder.DateRange24h, "json")
	if err != nil {
		return nil, fmt.Errorf("failed to generate report url: %v", err)
	}
	log.Printf("fetching report %q", reportJSONURL)
	response, err := http.DefaultClient.Get(reportJSONURL)
	if err != nil {
		return nil, fmt.Errorf("error fetching report %q", reportJSONURL)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error %s fetching report %q", response.Status, reportJSONURL)
	}
	defer response.Body.Close()

	var flakefinderReportData flakefinder.Params
	err = json.NewDecoder(response.Body).Decode(&flakefinderReportData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode flakefinder json from %s: %v", reportJSONURL, err)
	}
	return &flakefinderReportData, nil
}

func aggregateTopXTests(recentFlakeFinderReports []*flakefinder.Params) TopXTests {

	// store test names for quarantined tests to later on mark the displayed results
	// since we want to aggregate over a test that has a changing quarantine status during the period
	quarantinedTestNames := map[string]struct{}{}

	testNamesByTopXTests := map[string]*TopXTest{}
	for _, reportData := range recentFlakeFinderReports {
		for i := 0; i < len(reportData.Tests); i++ {

			// the original test name is used to retrieve the test data from the flakefinder report
			// i.e. all access to `reportData`
			originalTestName := reportData.Tests[i]

			// while the normalized test name is used to aggregate the test data for the stats report
			// i.e. `testNamesByTopXTests`
			normalizedTestName := normalizeTestName(originalTestName)

			if isQuarantineLabelPresent(originalTestName) {
				quarantinedTestNames[normalizedTestName] = struct{}{}
			}

			aggregateFailuresPerJob(reportData, originalTestName, testNamesByTopXTests, normalizedTestName)
		}
	}

	markQuarantinedTests(testNamesByTopXTests, quarantinedTestNames)

	return generateSortedAllTests(testNamesByTopXTests)
}

func aggregateFailuresPerJob(reportData *flakefinder.Params, originalTestName string, testNamesByTopXTests map[string]*TopXTest, normalizedTestName string) {
	for jobName, jobFailures := range reportData.Data[originalTestName] {

		if jobFailures.Failed == 0 {
			continue
		}

		currentTopXTest, topXTestsExists := testNamesByTopXTests[normalizedTestName]
		if !topXTestsExists {
			testNamesByTopXTests[normalizedTestName] = NewTopXTest(normalizedTestName)
		}
		currentTopXTest = testNamesByTopXTests[normalizedTestName]

		aggregateAllFailuresPerTest(currentTopXTest, jobFailures)
		aggregateFailuresPerTestPerDay(currentTopXTest, reportData, jobFailures)
		aggregateFailuresPerTestPerLane(currentTopXTest, jobName, jobFailures)
	}
}

func aggregateAllFailuresPerTest(currentTopXTest *TopXTest, jobFailures *flakefinder.Details) {
	currentTopXTest.AllFailures.add(jobFailures.Failed)
}

func aggregateFailuresPerTestPerDay(currentTopXTest *TopXTest, reportData *flakefinder.Params, jobFailures *flakefinder.Details) {
	_, failuresPerDayExists := currentTopXTest.FailuresPerDay[reportData.StartOfReport]
	if !failuresPerDayExists {
		date := formatFromRFC3339ToRFCDate(reportData.StartOfReport)
		currentTopXTest.FailuresPerDay[reportData.StartOfReport] = &FailureCounter{
			Name: formatFromSourceToTargetFormat(reportData.StartOfReport, time.RFC3339, dayFormat),
			URL:  fmt.Sprintf("https://storage.googleapis.com/kubevirt-prow/reports/flakefinder/kubevirt/kubevirt/flakefinder-%s-024h.html", date),
		}
	}
	currentTopXTest.FailuresPerDay[reportData.StartOfReport].add(jobFailures.Failed)
}

func aggregateFailuresPerTestPerLane(currentTopXTest *TopXTest, jobName string, jobFailures *flakefinder.Details) {
	_, failuresPerLaneExists := currentTopXTest.FailuresPerLane[jobName]
	if !failuresPerLaneExists {
		currentTopXTest.FailuresPerLane[jobName] = &FailureCounter{
			Name: jobName,
			URL:  generateTestGridURLForJob(jobName),
		}
	}
	currentTopXTest.FailuresPerLane[jobName].add(jobFailures.Failed)
}

func generateTestGridURLForJob(jobName string) string {
	switch {
	case strings.HasPrefix(jobName, "pull"):
		return fmt.Sprintf("https://testgrid.k8s.io/kubevirt-presubmits#%s&width=20", jobName)
	case strings.HasPrefix(jobName, "periodic"):
		return fmt.Sprintf("https://testgrid.k8s.io/kubevirt-periodics#%s&width=20", jobName)
	default:
		panic(fmt.Errorf("no case for jobName %q", jobName))
	}
}

func markQuarantinedTests(testNamesByTopXTests map[string]*TopXTest, quarantinedTestNames map[string]struct{}) {
	for _, test := range testNamesByTopXTests {
		if _, wasQuarantinedDuringReportRange := quarantinedTestNames[test.Name]; wasQuarantinedDuringReportRange {
			test.NoteHasBeenQuarantined = true
		}
	}
}

func generateSortedAllTests(testNamesByTopXTests map[string]*TopXTest) TopXTests {
	var allTests TopXTests
	for _, test := range testNamesByTopXTests {
		allTests = append(allTests, test)
	}
	sort.Sort(allTests)
	return allTests
}

func writeReport(overallFailures *TopXTest, allTests TopXTests) error {
	htmlReportOutputWriter, err := os.Create(opts.outputFile)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", opts.outputFile, err)
	}
	log.Printf("Writing html to %q", opts.outputFile)
	defer htmlReportOutputWriter.Close()

	templateData := &TemplateData{
		OverallFailures: overallFailures,
		TopXTests:       allTests,
		DaysInThePast:   opts.daysInThePast,
		Date:            time.Now(),
		ShareCategories: shareCategories,
		Org:             opts.org,
		Repo:            opts.repo,
	}
	err = flakefinder.WriteTemplateToOutput(htmlTemplate, templateData, htmlReportOutputWriter)
	if err != nil {
		return fmt.Errorf("failed to write to file %q: %w", opts.outputFile, err)
	}
	return nil
}

func formatFromRFC3339ToRFCDate(date string) string {
	return formatFromSourceToTargetFormat(date, time.RFC3339, rfc3339Date)
}

func formatFromSourceToTargetFormat(dayDate, sourceFormat, targetFormat string) string {
	date, err := time.Parse(sourceFormat, dayDate)
	if err != nil {
		panic(err)
	}
	return date.Format(targetFormat)
}

func isQuarantineLabelPresent(testName string) bool {
	return strings.Contains(testName, "[QUARANTINE]")
}

// normalizeTestName removes quarantine label and in that process eventually multiple spaces to have a chance to find
// the test name again. However, for bigger renamings it can't do much.
func normalizeTestName(testName string) string {
	return multipleSpacesRegex.ReplaceAllString(strings.Replace(testName, "[QUARANTINE]", "", -1), " ")
}

func NewTopXTest(topXTestName string) *TopXTest {
	return &TopXTest{
		Name:            topXTestName,
		AllFailures:     &FailureCounter{Name: "All failures"},
		FailuresPerDay:  map[string]*FailureCounter{},
		FailuresPerLane: map[string]*FailureCounter{},
	}
}

func previousDay(now time.Time) time.Time {
	return now.Add(-24 * time.Hour)
}

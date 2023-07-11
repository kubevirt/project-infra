package main

import (
	_ "embed"
	"encoding/json"
	"flag"
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
	iAllFailures := t[i].AllFailures
	jAllFailures := t[j].AllFailures
	return iAllFailures.Sum > jAllFailures.Sum ||
		(iAllFailures.Sum == jAllFailures.Sum && iAllFailures.Max > jAllFailures.Max) ||
		(iAllFailures.Sum == jAllFailures.Sum && iAllFailures.Max == jAllFailures.Max && iAllFailures.Avg > jAllFailures.Avg)
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
				overall.FailuresPerDay[day] = &FailureCounter{Name: failuresPerDay.Name}
			}
			overall.FailuresPerDay[day].add(failuresPerDay.Sum)
		}

		// aggregate failures per test per lane
		for lane, failuresPerLane := range test.FailuresPerLane {
			_, failuresPerLaneExists := overall.FailuresPerLane[lane]
			if !failuresPerLaneExists {
				overall.FailuresPerLane[lane] = &FailureCounter{Name: lane, ShowURL: true}
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

func (t TopXTests) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
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
	ShowURL       bool
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

var opts = options{}

const defaultDaysInThePast = 14
const defaultOrg = "kubevirt"
const defaultRepo = "kubevirt"

func main() {

	flag.IntVar(&opts.daysInThePast, "days-in-the-past", defaultDaysInThePast, "determines how much days in the past till today are covered")
	flag.StringVar(&opts.outputFile, "output-file", "", "outputfile to write to, default is a tempfile in folder")
	flag.BoolVar(&opts.overwriteOutputFile, "overwrite-output-file", false, "whether outputfile is set to be overwritten if it exists")
	flag.StringVar(&opts.org, "org", defaultOrg, "GitHub org to use for fetching report data from gcs dir")
	flag.StringVar(&opts.repo, "repo", defaultRepo, "GitHub repo to use for fetching report data from gcs dir")
	flag.Parse()

	if opts.daysInThePast <= 0 {
		log.Fatalf("invalid value for daysInThePast %d", opts.daysInThePast)
	}
	if opts.outputFile == "" {
		file, err := os.CreateTemp("", "flake-stats-*.html")
		if err != nil {
			log.Fatalf("failed to generate temp file: %v", err)
		}
		opts.outputFile = file.Name()
	} else {
		if !opts.overwriteOutputFile {
			stats, err := os.Stat(opts.outputFile)
			if stats != nil || !os.IsNotExist(err) {
				log.Fatalf("file exists: %v", err)
			}
		}
	}

	targetReportDate := previousDay(time.Now())
	var recentFlakefinderReports []flakefinder.Params
	for i := 0; i < opts.daysInThePast; i++ {
		reportJSONURL, err := flakefinder.GenerateReportURL(opts.org, opts.repo, targetReportDate, flakefinder.DateRange24h, "json")
		if err != nil {
			log.Fatalf("failed to generate report url: %v", err)
		}
		log.Printf("fetching report %q", reportJSONURL)
		response, err := http.DefaultClient.Get(reportJSONURL)
		if err != nil {
			log.Fatalf("error fetching report %q", reportJSONURL)
		}
		if response.StatusCode != http.StatusOK {
			log.Fatalf("error %s fetching report %q", response.Status, reportJSONURL)
		}
		defer response.Body.Close()

		var flakefinderReportData flakefinder.Params
		err = json.NewDecoder(response.Body).Decode(&flakefinderReportData)
		if err != nil {
			log.Fatalf("failed to decode flakefinder json from %s: %v", reportJSONURL, err)
		}
		recentFlakefinderReports = append(recentFlakefinderReports, flakefinderReportData)

		targetReportDate = previousDay(targetReportDate)
	}

	// store test names for quarantined tests to later on mark the displayed results
	// since we want to aggregate over a test that has a changing quarantine status during the period
	quarantinedTestNames := map[string]struct{}{}
	testNamesByTopXTests := map[string]*TopXTest{}
	for _, reportData := range recentFlakefinderReports {
		for i := 0; i < len(reportData.Tests); i++ {
			if isQuarantineLabelPresent(reportData.Tests[i]) {
				quarantinedTestNames[normalizeTestName(reportData.Tests[i])] = struct{}{}
			}

			// the original test name is used to retrieve the test data from the flakefinder report
			// i.e. all access to `reportData`
			originalTestName := reportData.Tests[i]

			// while the normalized test name is used to aggregate the test data for the stats report
			// i.e. `testNamesByTopXTests`
			normalizedTestName := normalizeTestName(originalTestName)

			for jobName, jobFailures := range reportData.Data[originalTestName] {

				if strings.Index(jobName, "periodic") == 0 {
					continue
				}

				if jobFailures.Failed == 0 {
					continue
				}

				currentTopXTest, topXTestsExists := testNamesByTopXTests[normalizedTestName]
				if !topXTestsExists {
					testNamesByTopXTests[normalizedTestName] = NewTopXTest(normalizedTestName)
				}
				currentTopXTest = testNamesByTopXTests[normalizedTestName]

				// aggregate all failures per test
				currentTopXTest.AllFailures.add(jobFailures.Failed)

				// aggregate failures per test per day
				_, failuresPerDayExists := currentTopXTest.FailuresPerDay[reportData.StartOfReport]
				if !failuresPerDayExists {
					currentTopXTest.FailuresPerDay[reportData.StartOfReport] = &FailureCounter{Name: formatToDay(reportData.StartOfReport)}
				}
				currentTopXTest.FailuresPerDay[reportData.StartOfReport].add(jobFailures.Failed)

				// aggregate failures per test per lane
				_, failuresPerLaneExists := currentTopXTest.FailuresPerLane[jobName]
				if !failuresPerLaneExists {
					currentTopXTest.FailuresPerLane[jobName] = &FailureCounter{Name: jobName, ShowURL: true}
				}
				currentTopXTest.FailuresPerLane[jobName].add(jobFailures.Failed)
			}
		}
	}

	var allTests TopXTests
	for _, test := range testNamesByTopXTests {
		if _, wasQuarantinedDuringReportRange := quarantinedTestNames[test.Name]; wasQuarantinedDuringReportRange {
			test.NoteHasBeenQuarantined = true
		}
		allTests = append(allTests, test)
	}
	sort.Sort(allTests)

	overallFailures := allTests.CalculateShareFromTotalFailures()

	htmlReportOutputWriter, err := os.Create(opts.outputFile)
	if err != nil {
		log.Fatalf("failed to write report %q: %v", opts.outputFile, err)
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
		log.Fatalf("Writing html to %q: %v", opts.outputFile, err)
	}

}

func formatToDay(dayDate string) string {
	date, err := time.Parse(time.RFC3339, dayDate)
	if err != nil {
		panic(err)
	}
	return date.Format("Mon, 02 Jan 2006")
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

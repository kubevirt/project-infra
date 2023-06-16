package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"kubevirt.io/project-infra/robots/pkg/flakefinder"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

//go:embed flake-stats.gohtml
var htmlTemplate string

type TemplateData struct {
	TopXTests
	DaysInThePast int
	Date          time.Time
}

type TopXTests []*TopXTest

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

type FailureCounter struct {
	Name  string
	Count int
	Sum   int
	Avg   float64
	Max   int
}

func (c *FailureCounter) add(value int) {
	c.Sum += value
	if value > c.Max {
		c.Max = value
	}
	c.Avg = (float64(value) + float64(c.Count)*c.Avg) / float64(c.Count+1)
	c.Count++
}

type options struct {
	daysInThePast       int
	outputFile          string
	overwriteOutputFile bool
}

var opts = options{}

const defaultDaysInThePast = 14

func main() {

	flag.IntVar(&opts.daysInThePast, "days-in-the-past", defaultDaysInThePast, "determines how much days in the past till today are covered")
	flag.StringVar(&opts.outputFile, "output-file", "", "outputfile to write to, default is a tempfile in folder")
	flag.BoolVar(&opts.overwriteOutputFile, "overwrite-output-file", false, "whether outputfile is set to be overwritten if it exists")
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
		reportJSONURL, err := flakefinder.GenerateReportURL("kubevirt", "kubevirt", targetReportDate, flakefinder.DateRange24h, "json")
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

	quarantinedTestNames := map[string]struct{}{}
	testNamesByTopXTests := map[string]*TopXTest{}
	for _, reportData := range recentFlakefinderReports {
		for i := 0; i < len(reportData.Tests); i++ {

			topXTestName := reportData.Tests[i]
			if strings.Contains(topXTestName, "[QUARANTINE]") {
				// store the test name to later on filter the displayed results
				testNameWithoutQuarantineLabel := strings.Replace(topXTestName, "[QUARANTINE]", "", -1)
				quarantinedTestNames[testNameWithoutQuarantineLabel] = struct{}{}
				topXTestName = testNameWithoutQuarantineLabel
			}

			currentTopXTest, topXTestsExists := testNamesByTopXTests[topXTestName]
			if !topXTestsExists {
				testNamesByTopXTests[topXTestName] = NewTopXTest(topXTestName)
				currentTopXTest = testNamesByTopXTests[topXTestName]
			}

			for jobName, jobFailures := range reportData.Data[topXTestName] {

				if jobFailures.Failed == 0 {
					continue
				}

				if strings.Index(jobName, "periodic") == 0 {
					continue
				}

				// aggregate all failures per test
				currentTopXTest.AllFailures.add(jobFailures.Failed)

				// aggregate failures per test per day
				_, failuresPerDayExists := currentTopXTest.FailuresPerDay[reportData.StartOfReport]
				if !failuresPerDayExists {
					currentTopXTest.FailuresPerDay[reportData.StartOfReport] = &FailureCounter{Name: strings.Replace(reportData.StartOfReport, "T00:00:00Z", "", -1)}
				}
				currentTopXTest.FailuresPerDay[reportData.StartOfReport].add(jobFailures.Failed)

				// aggregate failures per test per lane
				_, failuresPerLaneExists := currentTopXTest.FailuresPerLane[jobName]
				if !failuresPerLaneExists {
					currentTopXTest.FailuresPerLane[jobName] = &FailureCounter{Name: jobName}
				}
				currentTopXTest.FailuresPerLane[jobName].add(jobFailures.Failed)
			}
		}
	}

	var allTests TopXTests
	for _, test := range testNamesByTopXTests {
		_, wasQuarantined := quarantinedTestNames[test.Name]
		test.NoteHasBeenQuarantined = wasQuarantined
		allTests = append(allTests, test)
	}
	sort.Sort(allTests)

	htmlReportOutputWriter, err := os.Create(opts.outputFile)
	if err != nil {
		log.Fatalf("failed to write report %q: %v", opts.outputFile, err)
	}
	log.Printf("Writing html to %q", opts.outputFile)
	defer htmlReportOutputWriter.Close()

	templateData := &TemplateData{
		TopXTests:     allTests,
		DaysInThePast: opts.daysInThePast,
		Date:          time.Now(),
	}
	err = flakefinder.WriteTemplateToOutput(htmlTemplate, templateData, htmlReportOutputWriter)
	if err != nil {
		log.Fatalf("Writing html to %q: %v", opts.outputFile, err)
	}

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

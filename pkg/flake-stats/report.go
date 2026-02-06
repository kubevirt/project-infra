/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright the KubeVirt Authors.
 *
 */

package flakestats

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"kubevirt.io/project-infra/pkg/flakefinder"

	"github.com/sirupsen/logrus"
)

//go:embed flake-stats.gohtml
var htmlTemplate string

//go:embed flake-stats.gomd
var mdTemplate string

const defaultDaysInThePast = 14
const defaultOrg = "kubevirt"
const defaultRepo = "kubevirt"
const dayFormat = "Mon, 02 Jan 2006"

const defaultOutputFormatHTML = "html"
const outputFormatJSON = "json"
const outputFormatMD = "md"

var outputFormats = []string{
	defaultOutputFormatHTML,
	outputFormatJSON,
	outputFormatMD,
}

func GenerateReport(options *Options) error {
	return NewFlakeStatsReport(options).generate()
}

func NewFlakeStatsReport(options *Options) FlakeStats {
	return FlakeStats{
		reportOpts: &options.ReportOptions,
		writeOpts:  &options.WriteOptions,
	}
}

func NewFlakeStatsAggregate(reportOptions *ReportOptions) FlakeStats {
	return FlakeStats{
		reportOpts: reportOptions,
	}
}

type FlakeStats struct {
	reportOpts *ReportOptions
	writeOpts  *WriteOptions
}

func (r FlakeStats) generate() error {
	topXTests, err := r.AggregateData()
	if err != nil {
		return err
	}
	shareFromTotalFailures := topXTests.CalculateShareFromTotalFailures()
	switch r.writeOpts.OutputFormat {
	case defaultOutputFormatHTML:
		err = r.writeHTMLReport(shareFromTotalFailures, topXTests)
		if err != nil {
			return fmt.Errorf("failed writing html report: %w", err)
		}
	case outputFormatMD:
		err = r.writeMDReport(shareFromTotalFailures, topXTests)
		if err != nil {
			return fmt.Errorf("failed writing markdown report: %w", err)
		}
	case outputFormatJSON:
		var jsonOutput []byte
		jsonOutput, err = json.Marshal(topXTests)
		if err != nil {
			return fmt.Errorf("failed marshalling data: %w", err)
		}
		err = os.WriteFile(r.writeOpts.OutputFile, jsonOutput, 0666)
		if err != nil {
			return fmt.Errorf("failed marshalling data: %w", err)
		}
	}
	return nil
}

// AggregateData fetches the 24h flakefinder report data for the days given in the Options and returns a TopXTests
// that holds the aggregated data for all the tests encountered.
// err will be non nil if an error has been encountered while fetching the flakefinder reports.
func (r FlakeStats) AggregateData() (TopXTests, error) {
	recentFlakeFinderReports, err := r.fetchFlakeFinder24hReportsForRecentDays()
	if err != nil {
		return nil, fmt.Errorf("failed fetching flake reports: %w", err)
	}
	topXTests := r.aggregateTopXTests(recentFlakeFinderReports)
	return topXTests, nil
}

func (r FlakeStats) fetchFlakeFinder24hReportsForRecentDays() ([]*flakefinder.Params, error) {
	var recentFlakeFinderReports []*flakefinder.Params
	targetReportDate := previousDay(time.Now())
	for i := 0; i < r.reportOpts.DaysInThePast; i++ {
		flakeFinderReportData, err := r.fetchFlakeFinder24hReportData(targetReportDate)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve flakefinder report data for %v: %v", targetReportDate, err)
		}
		recentFlakeFinderReports = append(recentFlakeFinderReports, flakeFinderReportData)

		targetReportDate = previousDay(targetReportDate)
	}
	return recentFlakeFinderReports, nil
}

func (r FlakeStats) fetchFlakeFinder24hReportData(targetReportDate time.Time) (*flakefinder.Params, error) {
	reportJSONURL, err := flakefinder.GenerateReportURL(r.reportOpts.Org, r.reportOpts.Repo, targetReportDate, flakefinder.DateRange24h, "json")
	if err != nil {
		return nil, fmt.Errorf("failed to generate report url: %v", err)
	}
	logrus.Printf("fetching report %q", reportJSONURL)
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

func (r FlakeStats) aggregateTopXTests(recentFlakeFinderReports []*flakefinder.Params) TopXTests {

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
			normalizedTestName := flakefinder.NormalizeTestName(originalTestName)

			if flakefinder.IsQuarantineLabelPresent(originalTestName) {
				quarantinedTestNames[normalizedTestName] = struct{}{}
			}

			r.aggregateFailuresPerJob(reportData, originalTestName, testNamesByTopXTests, normalizedTestName)
		}
	}

	markQuarantinedTests(testNamesByTopXTests, quarantinedTestNames)

	return r.generateSortedAllTests(testNamesByTopXTests)
}

func (r FlakeStats) aggregateFailuresPerJob(reportData *flakefinder.Params, originalTestName string, testNamesByTopXTests map[string]*TopXTest, normalizedTestName string) {
	for jobName, jobFailures := range reportData.Data[originalTestName] {

		if r.reportOpts.filterLaneRegex != nil && r.reportOpts.filterLaneRegex.MatchString(jobName) {
			continue
		}

		if r.reportOpts.matchingLaneRegex != nil && !r.reportOpts.matchingLaneRegex.MatchString(jobName) {
			continue
		}

		if r.reportOpts.FilterPeriodicJobRunResults && strings.Index(jobName, "periodic") == 0 {
			continue
		}

		if jobFailures.Failed == 0 {
			continue
		}

		_, topXTestsExists := testNamesByTopXTests[normalizedTestName]
		if !topXTestsExists {
			testNamesByTopXTests[normalizedTestName] = NewTopXTest(normalizedTestName)
		}
		currentTopXTest := testNamesByTopXTests[normalizedTestName]

		r.aggregateAllFailuresPerTest(currentTopXTest, jobFailures)
		r.aggregateFailuresPerTestPerDay(currentTopXTest, reportData, jobFailures)
		r.aggregateFailuresPerTestPerLane(currentTopXTest, jobName, jobFailures)
	}
}

func (r FlakeStats) aggregateAllFailuresPerTest(currentTopXTest *TopXTest, jobFailures *flakefinder.Details) {
	currentTopXTest.AllFailures.add(jobFailures.Failed)
}

func (r FlakeStats) aggregateFailuresPerTestPerDay(currentTopXTest *TopXTest, reportData *flakefinder.Params, jobFailures *flakefinder.Details) {
	fc, exists := currentTopXTest.FailuresPerDay[reportData.StartOfReport]
	if !exists {
		date := formatFromRFC3339ToRFCDate(reportData.StartOfReport)
		fc = &FailureCounter{
			Name:    formatFromSourceToTargetFormat(reportData.StartOfReport, time.RFC3339, dayFormat),
			URL:     fmt.Sprintf("https://storage.googleapis.com/kubevirt-prow/reports/flakefinder/kubevirt/kubevirt/flakefinder-%s-024h.html", date),
			PrCount: len(reportData.PrNumbers),
		}
		currentTopXTest.FailuresPerDay[reportData.StartOfReport] = fc
	} else {
		fc.PrCount += len(reportData.PrNumbers)
	}
	fc.add(jobFailures.Failed)
}

func (r FlakeStats) aggregateFailuresPerTestPerLane(currentTopXTest *TopXTest, jobName string, jobFailures *flakefinder.Details) {
	_, failuresPerLaneExists := currentTopXTest.FailuresPerLane[jobName]
	if !failuresPerLaneExists {
		currentTopXTest.FailuresPerLane[jobName] = &FailureCounter{
			Name: jobName,
			URL:  generateTestGridURLForJob(jobName),
		}
	}
	currentTopXTest.FailuresPerLane[jobName].add(jobFailures.Failed)
}

func (r FlakeStats) generateSortedAllTests(testNamesByTopXTests map[string]*TopXTest) TopXTests {
	var allTests TopXTests
	for _, test := range testNamesByTopXTests {
		test.daysInThePast = r.reportOpts.DaysInThePast
		allTests = append(allTests, test)
	}
	sort.Sort(allTests)
	return allTests
}

func (r FlakeStats) writeHTMLReport(overallFailures *TopXTest, topXTests TopXTests) error {
	htmlReportOutputWriter, err := os.Create(r.writeOpts.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", r.writeOpts.OutputFile, err)
	}
	logrus.Printf("Writing html to %q", r.writeOpts.OutputFile)
	defer htmlReportOutputWriter.Close()

	templateData := &ReportData{
		OverallFailures: overallFailures,
		TopXTests:       topXTests,
		DaysInThePast:   r.reportOpts.DaysInThePast,
		Date:            time.Now(),
		ShareCategories: shareCategories,
		Org:             r.reportOpts.Org,
		Repo:            r.reportOpts.Repo,
	}
	err = flakefinder.WriteTemplateToOutput(htmlTemplate, templateData, htmlReportOutputWriter)
	if err != nil {
		return fmt.Errorf("failed to write to file %q: %w", r.writeOpts.OutputFile, err)
	}
	return nil
}

func (r FlakeStats) writeMDReport(overallFailures *TopXTest, topXTests TopXTests) error {
	mdReportOutputWriter, err := os.Create(r.writeOpts.OutputFile)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", r.writeOpts.OutputFile, err)
	}
	logrus.Printf("Writing markdown to %q", r.writeOpts.OutputFile)
	defer mdReportOutputWriter.Close()

	templateData := &ReportData{
		OverallFailures: overallFailures,
		TopXTests:       topXTests,
		DaysInThePast:   r.reportOpts.DaysInThePast,
		Date:            time.Now(),
		ShareCategories: shareCategories,
		Org:             r.reportOpts.Org,
		Repo:            r.reportOpts.Repo,
	}
	err = flakefinder.WriteTemplateToOutput(mdTemplate, templateData, mdReportOutputWriter)
	if err != nil {
		return fmt.Errorf("failed to write to file %q: %w", r.writeOpts.OutputFile, err)
	}
	return nil
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

func formatFromRFC3339ToRFCDate(date string) string {
	return formatFromSourceToTargetFormat(date, time.RFC3339, time.DateOnly)
}

func formatFromSourceToTargetFormat(dayDate, sourceFormat, targetFormat string) string {
	date, err := time.Parse(sourceFormat, dayDate)
	if err != nil {
		panic(err)
	}
	return date.Format(targetFormat)
}

func previousDay(now time.Time) time.Time {
	return now.Add(-24 * time.Hour)
}

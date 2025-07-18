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

package cmd

import (
	_ "embed"
	"fmt"
	"github.com/joshdk/go-junit"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"html/template"
	"io"
	flakestats "kubevirt.io/project-infra/robots/pkg/flake-stats"
	"kubevirt.io/project-infra/robots/pkg/options"
	"kubevirt.io/project-infra/robots/pkg/searchci"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
)

const (
	shortReportHelp = `Creates a report of the most flaky tests`
)

var (
	mostFlakyTestsReportCmd = &cobra.Command{
		Use:   "report",
		RunE:  MostFlakyTestsReport,
		Short: shortReportHelp,
		Long: shortReportHelp + ` using data from flake-stats and search.ci

It fetches data for flake-stats from the last x days, then for each test it
scrapes search.ci for impact on lanes.
All tests that exceed either the 3 day or 14 day value are added to
the report.
All output is aggregated with links to sources into an html page.
`,
	}

	//go:embed most-flaky-tests-report.gohtml
	mostFlakyTestsReportTemplate string

	outputFileOpts = options.NewOutputFileOptions("most-flaky-tests-*.html")
)

func init() {
	mostFlakyTestsReportCmd.PersistentFlags().IntVar(&quarantineOpts.daysInThePast, "days-in-the-past", 14, "the number of days in the past")
	mostFlakyTestsReportCmd.PersistentFlags().StringVar(&outputFileOpts.OutputFile, "output-file", "", "the name of the output file, or empty string to create a temp file")
	mostFlakyTestsReportCmd.PersistentFlags().BoolVar(&outputFileOpts.OverwriteOutputFile, "overwrite-output-file", false, "whether to overwrite the output file")
	mostFlakyTestsReportCmd.PersistentFlags().BoolVar(&quarantineOpts.filterPeriodicJobRunResults, "filter-periodic-job-run-results", true, "whether to filter the results for periodics")
}

var sigMatcher = regexp.MustCompile(`\[(sig-[^]]+)]`)

func MostFlakyTestsReport(_ *cobra.Command, _ []string) error {
	reportOpts := flakestats.NewDefaultReportOpts(
		flakestats.DaysInThePast(quarantineOpts.daysInThePast),
		flakestats.FilterPeriodicJobRunResults(quarantineOpts.filterPeriodicJobRunResults),
		flakestats.FilterLaneRegex("rehearsal|e2e.*arm"),
	)
	err := reportOpts.Validate()
	if err != nil {
		return err
	}
	err = outputFileOpts.Validate()
	if err != nil {
		return err
	}
	topXTests, err := flakestats.NewFlakeStatsAggregate(reportOpts).AggregateData()
	if err != nil {
		return fmt.Errorf("error while aggregating data: %w", err)
	}
	sigs, mostFlakyTestsBySig, err := aggregateMostFlakyTestsBySIG(topXTests)
	if err != nil {
		return err
	}

	reportTemplate, err := template.New("mostFlakyTests").Parse(mostFlakyTestsReportTemplate)
	if err != nil {
		return fmt.Errorf("could not read template: %w", err)
	}
	outputFile, err := os.Create(outputFileOpts.OutputFile)
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}
	err = reportTemplate.Execute(outputFile, NewMostFlakyTestsTemplateData(mostFlakyTestsBySig, sigs))
	if err != nil {
		return fmt.Errorf("could not execute template: %w", err)
	}
	log.Infof("report written to %q", outputFile.Name())
	return nil
}

const noSIGKey = "NONE"

func aggregateMostFlakyTestsBySIG(topXTests flakestats.TopXTests) (sigs []string, mostFlakyTestsBySIG map[string]map[searchci.TimeRange][]*TestToQuarantine, err error) {
	mostFlakyTests := make(map[searchci.TimeRange][]*TestToQuarantine)
	for _, timeRange := range mostFlakyTestsTimeRanges {
		for _, topXTest := range topXTests {
			candidate, err := getQuarantineCandidate(topXTest, timeRange)
			if err != nil {
				return nil, nil, fmt.Errorf("could not scrape results for Test %q: %w", topXTest.Name, err)
			}
			if candidate == nil {
				continue
			}
			mostFlakyTests[timeRange] = append(mostFlakyTests[timeRange], candidate)
		}
		sortTestToQuarantineFunc := func(i, j int) bool {
			return mostFlakyTests[timeRange][i].RelevantImpacts[0].Percent > mostFlakyTests[timeRange][j].RelevantImpacts[0].Percent
		}
		sort.Slice(mostFlakyTests[timeRange], sortTestToQuarantineFunc)
	}
	mostFlakyTestsBySIG = make(map[string]map[searchci.TimeRange][]*TestToQuarantine)
	mapOfSIGs := make(map[string]struct{})
	for timeRange, testsToQuarantine := range mostFlakyTests {
		for _, testToQuarantine := range testsToQuarantine {
			key := noSIGKey
			if sigMatcher.MatchString(testToQuarantine.Test.Name) {
				submatch := sigMatcher.FindStringSubmatch(testToQuarantine.Test.Name)
				key = submatch[1]
			}
			mapOfSIGs[key] = struct{}{}
			if _, ok := mostFlakyTestsBySIG[key]; !ok {
				mostFlakyTestsBySIG[key] = make(map[searchci.TimeRange][]*TestToQuarantine)
			}
			mostFlakyTestsBySIG[key][timeRange] = append(mostFlakyTestsBySIG[key][timeRange], testToQuarantine)
		}
	}
	for sig := range mapOfSIGs {
		sigs = append(sigs, sig)
	}
	sort.Slice(sigs, func(i, j int) bool {
		if sigs[i] == noSIGKey || sigs[j] == noSIGKey {
			return sigs[j] == noSIGKey
		}
		return sigs[i] < sigs[j]
	})
	return sigs, mostFlakyTestsBySIG, nil
}

func getQuarantineCandidate(topXTest *flakestats.TopXTest, timeRange searchci.TimeRange) (*TestToQuarantine, error) {
	impacts, err := searchci.ScrapeImpacts(topXTest.Name, timeRange)
	if err != nil {
		return nil, fmt.Errorf("could not scrape results for test %q: %w", topXTest.Name, err)
	}
	if impacts == nil {
		log.Infof("search.ci scrape found no matches for test %q", topXTest.Name)
		return nil, nil
	}
	impacts = searchci.FilterImpactsBy(impacts,
		matchesAnyFailureLane(topXTest),
		isNotARehearsal(),
		isNotAFlakeCheckRun(),
		isNotADeQuarantineCheckRun(),
		hasNotOnlyClusteredFailures(),
	)
	if impacts == nil {
		log.Infof("search.ci filter left no matches for test %q", topXTest.Name)
		return nil, nil
	}
	sort.Slice(impacts, func(i, j int) bool {
		return impacts[i].Percent > impacts[j].Percent
	})
	newTestToQuarantine := &TestToQuarantine{
		Test:            topXTest,
		RelevantImpacts: impacts,
		SearchCIURL:     searchci.NewScrapeURL(topXTest.Name, timeRange),
		TimeRange:       timeRange,
	}
	return newTestToQuarantine, nil
}

var basePartURLMatcher = regexp.MustCompile(`.*(kubevirt-prow/.*)`)

func hasNotOnlyClusteredFailures() searchci.FilterOpt {
	return func(i searchci.Impact) bool {
		for _, buildURL := range i.BuildURLs {
			basePartURL := basePartURLMatcher.FindStringSubmatch(buildURL.URL)[1]
			junitXMLURL := fmt.Sprintf("https://storage.googleapis.com/%s/artifacts/junit.functest.xml", basePartURL)
			junitXMLHTTPResponse, err := http.Get(junitXMLURL)
			if err != nil {
				log.Fatalf("failed to get junit xml from %q", junitXMLURL)
			}
			if junitXMLHTTPResponse.StatusCode != 200 {
				log.Fatalf("failed to get junit xml from %q", junitXMLURL)
			}
			defer junitXMLHTTPResponse.Body.Close()
			junitXML, err := io.ReadAll(junitXMLHTTPResponse.Body)
			if err != nil {
				log.Fatalf("failed to get junit xml from %q", junitXMLURL)
			}
			testSuites, err := junit.Ingest(junitXML)
			if err != nil {
				log.Fatalf("failed to get junit xml from %q", junitXMLURL)
			}
			for _, suite := range testSuites {
				if suite.Totals.Failed < 5 {
					return true
				}
			}
		}
		return false
	}
}

func matchesAnyFailureLane(topXTest *flakestats.TopXTest) func(i searchci.Impact) bool {
	var lanes []string
	for l := range topXTest.FailuresPerLane {
		lanes = append(lanes, l)
	}
	laneMatcher := regexp.MustCompile(fmt.Sprintf(`http.*/(%s)[^/]+[^/]+$`, strings.Join(lanes, "|")))
	laneMatcherOpt := func(i searchci.Impact) bool {
		return laneMatcher.MatchString(i.URL)
	}
	return laneMatcherOpt
}

func isNotARehearsal() func(i searchci.Impact) bool {
	return func(i searchci.Impact) bool {
		return !strings.Contains(i.URL, "rehearsal")
	}
}

func isNotAFlakeCheckRun() func(i searchci.Impact) bool {
	return func(i searchci.Impact) bool {
		return !strings.Contains(i.URL, "check-tests-for-flakes")
	}
}

func isNotADeQuarantineCheckRun() func(i searchci.Impact) bool {
	return func(i searchci.Impact) bool {
		return !strings.Contains(i.URL, "check-dequarantine-test")
	}
}

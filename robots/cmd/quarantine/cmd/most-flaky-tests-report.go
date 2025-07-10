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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"html/template"
	flakestats "kubevirt.io/project-infra/robots/pkg/flake-stats"
	"kubevirt.io/project-infra/robots/pkg/options"
	"kubevirt.io/project-infra/robots/pkg/searchci"
	"os"
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
}

func MostFlakyTestsReport(_ *cobra.Command, _ []string) error {
	reportOpts := flakestats.NewDefaultReportOpts(
		flakestats.DaysInThePast(quarantineOpts.daysInThePast),
		flakestats.FilterPeriodicJobRunResults(true),
		flakestats.FilterLaneRegex("e2e.*arm|pull-kubevirt-check-tests-for-flakes.*|pull-kubevirt-check-dequarantine-Test.*"),
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
	mostFlakyTests := make(map[searchci.TimeRange][]*TestToQuarantine)
	for _, timeRange := range []searchci.TimeRange{searchci.FourteenDays, searchci.ThreeDays} {
		for _, topXTest := range topXTests {
			var candidate *TestToQuarantine
			candidate, err = getPossibleQuarantineCandidate(topXTest, timeRange)
			if err != nil {
				return fmt.Errorf("could not scrape results for Test %q: %w", topXTest.Name, err)
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

	reportTemplate, err := template.New("mostFlakyTests").Parse(mostFlakyTestsReportTemplate)
	if err != nil {
		return fmt.Errorf("could not read template: %w", err)
	}
	outputFile, err := os.Create(outputFileOpts.OutputFile)
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}
	err = reportTemplate.Execute(outputFile, NewMostFlakyTestsTemplateData(mostFlakyTests))
	if err != nil {
		return fmt.Errorf("could not execute template: %w", err)
	}
	log.Infof("report written to %q", outputFile.Name())
	return nil
}

func getPossibleQuarantineCandidate(topXTest *flakestats.TopXTest, timeRange searchci.TimeRange) (*TestToQuarantine, error) {
	relevantImpacts, err := searchci.ScrapeRelevantImpacts(topXTest.Name, timeRange)
	if err != nil {
		return nil, fmt.Errorf("could not scrape results for Test %q: %w", topXTest.Name, err)
	}
	relevantImpacts = searchci.FilterRelevantImpactsWithOpts(relevantImpacts, func(i searchci.Impact) bool {
		return !strings.Contains(i.URL, "pull-kubevirt-check-tests-for-flakes") &&
			!strings.Contains(i.URL, "pull-kubevirt-check-dequarantine-test")
	})
	if relevantImpacts == nil {
		log.Infof("search.ci found no matches for %q", topXTest.Name)
		return nil, nil
	}
	sortRelevantImpactsFunc := func(i, j int) bool {
		a, b := relevantImpacts[i], relevantImpacts[j]
		return a.Percent > b.Percent
	}
	sort.Slice(relevantImpacts, sortRelevantImpactsFunc)
	newTestToQuarantine := &TestToQuarantine{
		Test:            topXTest,
		RelevantImpacts: relevantImpacts,
		SearchCIURL:     searchci.NewScrapeURL(topXTest.Name, timeRange),
		TimeRange:       timeRange,
	}
	return newTestToQuarantine, nil
}

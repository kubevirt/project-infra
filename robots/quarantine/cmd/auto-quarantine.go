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
	"github.com/onsi/ginkgo/v2/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
	flakestats "kubevirt.io/project-infra/pkg/flake-stats"
	"kubevirt.io/project-infra/pkg/ginkgo"
	"kubevirt.io/project-infra/pkg/options"
	"kubevirt.io/project-infra/pkg/searchci"
	"os"
	"regexp"
	"strings"
	"text/template"
)

const (
	defaultMatchingLaneRegexString = `^pull-.*-sig-(compute(-serial|-migrations|-arm64)?|network|storage|operator)%s$`

	shortAutoTest = "Quarantines flaky tests matching the required criteria"
	longAutoTest  = shortAutoTest + `.

Uses the data from the flake stats, combines that with
scraped results from search.ci and then modifies all tests matching quarantine
criteria so that they are recognized by automation as being quarantined.`
)

var (
	autoQuarantineCmd = &cobra.Command{
		Use:   "auto",
		Short: shortAutoTest,
		Long:  longAutoTest,
		RunE:  AutoQuarantine,
	}

	autoQuarantineOpts autoQuarantineOptions

	//go:embed auto-quarantine-pr-description.gomd
	autoQuarantinePRDescriptionTemplate string

	reportTemplate *template.Template
)

func init() {
	autoQuarantineCmd.PersistentFlags().IntVar(&quarantineOpts.daysInThePast, "days-in-the-past", 14, "the number of days in the past")
	autoQuarantineCmd.PersistentFlags().IntVar(&autoQuarantineOpts.maxTestsToQuarantine, "max-tests-to-quarantine", 1, "the overall number of tests that are going to be quarantined in one run")
	autoQuarantineCmd.PersistentFlags().StringVar(&autoQuarantineOpts.releaseLaneSuffix, "release-lane-suffix", "", "the suffix for the release lane to target (i.e. -1.7) or empty for targeting the main branch")
	autoQuarantineCmd.PersistentFlags().StringVar(&autoQuarantineOpts.matchingLaneRegexString, "matching-lane-regex", defaultMatchingLaneRegexString, "the regular expression that the lanes need to match - note that there's a suffix placeholder required")
	autoQuarantineOpts.prDescriptionOutputFileOpts = options.NewOutputFileOptions(
		"pr-description-*.md",
		func(o *options.OutputFileOptions) { o.OverwriteOutputFile = true },
	)
	autoQuarantineCmd.PersistentFlags().StringVar(&autoQuarantineOpts.prDescriptionOutputFileOpts.OutputFile, "pr-description-output-file", "", "the path to the output file to write the PR description into, or if unset a temp file will be generated")

	var err error
	reportTemplate, err = template.New("prDescription").Parse(autoQuarantinePRDescriptionTemplate)
	if err != nil {
		log.Fatalf("could not read template: %v", err)
	}
}

func AutoQuarantine(_ *cobra.Command, _ []string) error {
	err := autoQuarantineOpts.prDescriptionOutputFileOpts.Validate()
	if err != nil {
		return err
	}

	reportOpts := flakestats.NewDefaultReportOpts(
		flakestats.DaysInThePast(quarantineOpts.daysInThePast),
		flakestats.FilterPeriodicJobRunResults(true),
		// we only want to look at lanes targeting a specific branch here, so either the main branch (suffix is empty string) or a specific release like 1.7
		flakestats.MatchingLaneRegex(autoQuarantineOpts.MatchingLaneRegexString()),
	)
	err = reportOpts.Validate()
	if err != nil {
		return err
	}

	topXTests, err := flakestats.NewFlakeStatsAggregate(reportOpts).AggregateData()
	if err != nil {
		return fmt.Errorf("error while aggregating data: %w", err)
	}

	reports, _, err := ginkgo.DryRun(quarantineOpts.testSourcePath)
	if err != nil {
		return fmt.Errorf("could not fetch ginkgo test reports: %w", err)
	}
	if reports == nil {
		return fmt.Errorf("could not find ginkgo test reports: %w", err)
	}

	testsToQuarantine, err := determineTestsForQuarantine(topXTests, reports)
	if err != nil {
		return err
	}

	if len(testsToQuarantine) == 0 {
		log.Infof("no tests to quarantine")
		return nil
	}

	err = quarantineTests(testsToQuarantine)
	if err != nil {
		return err
	}

	testsPerSIG := groupTestsBySIG(testsToQuarantine)

	err = writePRDescriptionToFile(autoQuarantineOpts.prDescriptionOutputFileOpts.OutputFile, testsPerSIG)
	if err != nil {
		return err
	}

	return nil
}

func determineTestsForQuarantine(topXTests flakestats.TopXTests, reports []types.Report) ([]*TestToQuarantine, error) {
	var jobHistoryURLMatcher = regexp.MustCompile(autoQuarantineOpts.MatchingLaneRegexString())
	var ceilingForTestsToQuarantine = autoQuarantineOpts.maxTestsToQuarantine
	var testsToIgnore = []string{"AfterSuite"}
	var testsToQuarantine []*TestToQuarantine
	for _, topXTest := range topXTests {
		log.Infof("%s{Count: %d, Sum: %d, Avg: %f, Max: %d}", topXTest.Name, topXTest.AllFailures.Count, topXTest.AllFailures.Sum, topXTest.AllFailures.Avg, topXTest.AllFailures.Max)

		if slices.Contains(testsToIgnore, topXTest.Name) {
			log.Infof("Ignoring %q", topXTest.Name)
			continue
		}

		// Prepare to find the required data to modify the Test
		matchingSpecReport := ginkgo.GetSpecReportByTestName(reports, topXTest.Name)
		if matchingSpecReport == nil {
			log.Warnf("could not find file for %q by name", topXTest.Name)
			continue
		}

		// scrape impact from search.ci.kubevirt.io
		timeRange := searchci.FourteenDays
		relevantImpacts, err := searchci.ScrapeImpacts(topXTest.Name, timeRange)
		if err != nil {
			return nil, fmt.Errorf("could not scrape results for Test %q: %w", topXTest.Name, err)
		}
		if relevantImpacts == nil {
			log.Infof("search.ci found no matches for %q", topXTest.Name)
			continue
		}
		var filteredRelevantImpacts []searchci.Impact
		for _, i := range relevantImpacts {
			elements := strings.Split(i.URL, "/")
			if len(elements) == 0 {
				return nil, fmt.Errorf("no last element in job history url %q", i.URL)
			}
			lastElement := elements[len(elements)-1]
			if !jobHistoryURLMatcher.MatchString(lastElement) {
				continue
			}
			filteredRelevantImpacts = append(filteredRelevantImpacts, i)
		}
		if filteredRelevantImpacts == nil {
			log.Infof("search.ci found no matches in relevant jobs for %q", topXTest.Name)
			continue
		}

		newTestToQuarantine := &TestToQuarantine{
			Test:            topXTest,
			RelevantImpacts: filteredRelevantImpacts,
			SpecReport:      matchingSpecReport,
			SearchCIURL:     searchci.NewScrapeURL(topXTest.Name, timeRange),
			TimeRange:       timeRange,
		}
		testsToQuarantine = append(testsToQuarantine, newTestToQuarantine)

		if ceilingForTestsToQuarantine > 0 &&
			len(testsToQuarantine) >= ceilingForTestsToQuarantine {
			log.Infof("ceiling (%d) of tests to quarantine reached", ceilingForTestsToQuarantine)
			break
		}
	}
	return testsToQuarantine, nil
}

func quarantineTests(testsToQuarantine []*TestToQuarantine) error {
	var testsQuarantined []string
	for _, testToQuarantine := range testsToQuarantine {
		err := ginkgo.QuarantineTest(testToQuarantine.SpecReport)
		if err != nil {
			return fmt.Errorf("could not quarantine test %q: %w", testToQuarantine.SpecReport.FullText(), err)
		}
		testsQuarantined = append(testsQuarantined, testToQuarantine.SpecReport.FullText())
	}
	log.Infof("Tests quarantined:\n%s", strings.Join(testsQuarantined, "\n"))
	return nil
}

func groupTestsBySIG(testsToQuarantine []*TestToQuarantine) TestsPerSIG {
	testsPerSIG := TestsPerSIG{}
	sigLabelMatcher := ginkgo.NewRegexLabelMatcher(fmt.Sprintf(`^(%s)$`, strings.Join([]string{"sig-compute", "sig-storage", "sig-network", "sig-monitoring"}, "|")))
	for _, testToQuarantine := range testsToQuarantine {
		sigLabels := ginkgo.ExtractLabels(*testToQuarantine.SpecReport, sigLabelMatcher)
		var firstSIGLabel string
		if len(sigLabels) == 0 {
			firstSIGLabel = "undefined"
		} else {
			firstSIGLabel = sigLabels[0]
		}
		sigKey := strings.TrimPrefix(firstSIGLabel, "sig-")
		testsPerSIG[sigKey] = append(testsPerSIG[sigKey], testToQuarantine)
	}
	return testsPerSIG
}

func writePRDescriptionToFile(outputFileName string, testsPerSIG TestsPerSIG) error {
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("could not create file: %w", err)
	}
	defer func() {
		err2 := outputFile.Close()
		if err2 != nil {
			log.Errorf("failed to write output file: %v", err2)
		}
	}()
	err = reportTemplate.Execute(outputFile, testsPerSIG)
	if err != nil {
		return fmt.Errorf("could not execute template: %w", err)
	}
	log.Infof("report written to %q", outputFileName)
	return nil
}

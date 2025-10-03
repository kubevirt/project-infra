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
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"k8s.io/utils/strings/slices"
	flakestats "kubevirt.io/project-infra/robots/pkg/flake-stats"
	"kubevirt.io/project-infra/robots/pkg/ginkgo"
	"kubevirt.io/project-infra/robots/pkg/searchci"
	"os"
	"path/filepath"
	"strings"
)

var autoQuarantineCmd = &cobra.Command{
	Use:  "auto",
	RunE: AutoQuarantine,
}

func init() {
	autoQuarantineCmd.PersistentFlags().IntVar(&quarantineOpts.daysInThePast, "days-in-the-past", 14, "the number of days in the past")
}

func AutoQuarantine(_ *cobra.Command, _ []string) error {
	reportOpts := flakestats.NewDefaultReportOpts(
		flakestats.DaysInThePast(quarantineOpts.daysInThePast),
		flakestats.FilterPeriodicJobRunResults(true),
	)
	topXTests, err := flakestats.NewFlakeStatsAggregate(reportOpts).AggregateData()
	if err != nil {
		return fmt.Errorf("error while aggregating data: %w", err)
	}
	var testsToQuarantine []*TestToQuarantine
	reports, _, err := ginkgo.DryRun(quarantineOpts.testSourcePath)
	defer os.Remove(filepath.Join(quarantineOpts.testSourcePath, "junit.functest.xml"))
	if err != nil {
		return fmt.Errorf("could not fetch ginkgo test reports: %w", err)
	}
	if reports == nil {
		return fmt.Errorf("could not find ginkgo test reports: %w", err)
	}

	testsToIgnore := []string{"AfterSuite"}
	for _, topXTest := range topXTests {
		log.Infof("%s{Count: %d, Sum: %d, Avg: %f, Max: %d}", topXTest.Name, topXTest.AllFailures.Count, topXTest.AllFailures.Sum, topXTest.AllFailures.Avg, topXTest.AllFailures.Max)

		if slices.Contains(testsToIgnore, topXTest.Name) {
			log.Infof("Ignoring %q", topXTest.Name)
			continue
		}

		// Prepare to find the required data to modify the Test
		matchingSpecReport := ginkgo.GetMatchingSpecReport(reports, topXTest.Name)
		if matchingSpecReport == nil {
			log.Warnf("could not find file for %q by name", topXTest.Name)
			continue
		}

		// scrape impact from search.ci.kubevirt.io
		relevantImpacts, err := searchci.ScrapeImpacts(topXTest.Name, searchci.FourteenDays)
		if err != nil {
			return fmt.Errorf("could not scrape results for Test %q: %w", topXTest.Name, err)
		}
		if relevantImpacts == nil {
			log.Infof("search.ci found no matches for %q", topXTest.Name)
			continue
		}

		newTestToQuarantine := &TestToQuarantine{
			Test:            topXTest,
			RelevantImpacts: relevantImpacts,
			SpecReport:      matchingSpecReport,
		}
		testsToQuarantine = append(testsToQuarantine, newTestToQuarantine)
	}

	var testsQuarantined []string
	for _, testToQuarantine := range testsToQuarantine {
		err = ginkgo.QuarantineTestInFile(testToQuarantine.SpecReport)
		if err != nil {
			return fmt.Errorf("could not quarantine test %q: %w", testToQuarantine.SpecReport.FullText(), err)
		}
		testsQuarantined = append(testsQuarantined, testToQuarantine.SpecReport.FullText())
	}
	log.Infof("Tests quarantined:\n%s", strings.Join(testsQuarantined, "\n"))
	return nil
}

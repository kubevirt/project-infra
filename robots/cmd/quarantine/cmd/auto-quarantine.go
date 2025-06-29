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
	flakestats "kubevirt.io/project-infra/robots/pkg/flake-stats"
	"kubevirt.io/project-infra/robots/pkg/ginkgo"
	"kubevirt.io/project-infra/robots/pkg/searchci"
)

type testToQuarantine struct {
	test            *flakestats.TopXTest
	relevantImpacts []searchci.Impact
}

func (t testToQuarantine) String() string {
	return fmt.Sprintf("testToQuarantine{test: %+v, relevantImpacts: %+v}", t.test, t.relevantImpacts)
}

var autoQuarantineCmd = &cobra.Command{
	Use:  "auto",
	RunE: AutoQuarantine,
}

func init() {
	autoQuarantineCmd.PersistentFlags().IntVar(&quarantineOpts.daysInThePast, "days-in-the-past", 14, "the number of days in the past")
}

func AutoQuarantine(cmd *cobra.Command, args []string) error {
	reportOpts := flakestats.NewDefaultReportOpts(
		flakestats.DaysInThePast(quarantineOpts.daysInThePast),
		flakestats.FilterPeriodicJobRunResults(true),
	)
	topXTests, err := flakestats.NewFlakeStatsAggregate(reportOpts).AggregateData()
	if err != nil {
		return fmt.Errorf("error while aggregating data: %w", err)
	}
	var testsToQuarantine []*testToQuarantine
	for _, topXTest := range topXTests {
		log.Infof("%s{Count: %d, Sum: %d, Avg: %f, Max: %d}", topXTest.Name, topXTest.AllFailures.Count, topXTest.AllFailures.Sum, topXTest.AllFailures.Avg, topXTest.AllFailures.Max)

		// Prepare to find the required data to modify the test
		descriptor, _, err := ginkgo.FindFileAndDescriptor(quarantineOpts.testSourcePath, topXTest.Name)
		if err != nil {
			return fmt.Errorf("could not find file or descriptor for test %q: %w", topXTest.Name, err)
		}

		testSubstring := descriptor.OutlineNode().Text

		// scrape impact from search.ci.kubevirt.io
		relevantImpacts, err := searchci.ScrapeRelevantImpacts(testSubstring, searchci.FourteenDays)
		if err != nil {
			return fmt.Errorf("could not scrape results for test %q: %w", topXTest.Name, err)
		}
		if relevantImpacts == nil {
			log.Infof("search.ci found no matches for %q", testSubstring)
			continue
		}
		newTestToQuarantine := &testToQuarantine{
			test:            topXTest,
			relevantImpacts: relevantImpacts,
		}
		testsToQuarantine = append(testsToQuarantine, newTestToQuarantine)
	}
	log.Infof("testsToQuarantine: %+v", testsToQuarantine)
	return fmt.Errorf("TODO")
}

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
	"time"

	"github.com/onsi/ginkgo/v2/types"
	flakestats "kubevirt.io/project-infra/pkg/flake-stats"
	"kubevirt.io/project-infra/pkg/searchci"
)

const filterLaneRegexDefault = "rehearsal"

type autoQuarantineOptions struct {
	maxTestsToQuarantine int
}

type quarantineOptions struct {
	testSourcePath string

	// autoQuarantine
	daysInThePast int

	filterPeriodicJobRunResults bool
	filterLaneRegex             string

	testName string
}

type TestToQuarantine struct {
	Test            *flakestats.TopXTest
	TimeRange       searchci.TimeRange
	SearchCIURL     string
	RelevantImpacts []searchci.Impact
	SpecReport      *types.SpecReport
}

type TestsPerSIG map[string][]*TestToQuarantine

var (
	mostFlakyTestsTimeRanges = []searchci.TimeRange{searchci.ThreeDays, searchci.FourteenDays}
)

func (t TestToQuarantine) String() string {
	return fmt.Sprintf("TestToQuarantine{Test: %+v, SearchCIURL: %q, RelevantImpacts: %+v}", t.Test, t.SearchCIURL, t.RelevantImpacts)
}

func NewMostFlakyTestsTemplateData(mostFlakyTestsBySig map[string]map[string][]*TestToQuarantine, sigs []string, testNames []string) MostFlakyTestsTemplateData {
	return MostFlakyTestsTemplateData{
		ReportCreation:      time.Now(),
		MostFlakyTestsBySig: mostFlakyTestsBySig,
		SIGs:                sigs,
		TestNames:           testNames,
		TimeRanges:          mostFlakyTestsTimeRanges,
	}
}

type MostFlakyTestsTemplateData struct {
	ReportCreation      time.Time
	TimeRanges          []searchci.TimeRange
	MostFlakyTestsBySig map[string]map[string][]*TestToQuarantine
	SIGs                []string
	TestNames           []string
}

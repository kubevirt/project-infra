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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package junit_merge

import (
	"fmt"
	"github.com/joshdk/go-junit"
	"strings"
	"time"
)

func Merge(suitesOfSuites [][]junit.Suite) (mergedSuites []junit.Suite, hasConflicts bool) {
	conflicts := []string{}
	testsByName := map[string]junit.Test{}
	for _, suiteOfSuites := range suitesOfSuites {
		for _, suite := range suiteOfSuites {
			for _, test := range suite.Tests {
				if previous, exists := testsByName[test.Name]; exists {
					if previous.Status != "skipped" && test.Status != "skipped" {
						conflicts = append(conflicts, fmt.Sprintf("conflict: test executed more than once: %+v , %+v", previous, test))
					}
				}
				testsByName[test.Name] = test
			}
		}
	}
	allTests := []junit.Test{}
	testStatus := map[junit.Status]int{
		junit.StatusPassed: 0,
		junit.StatusSkipped: 0,
		junit.StatusFailed: 0,
		junit.StatusError: 0,
	}
	var totalExecutionTime time.Duration
	for _, test := range testsByName {
		allTests = append(allTests, test)
		testStatus[test.Status] = testStatus[test.Status] + 1
		totalExecutionTime += test.Duration
	}
	return []junit.Suite{
		junit.Suite{
			Name:       "Tests Suite (merged)",
			Package:    "",
			Properties: nil,
			Tests:      allTests,
			SystemOut:  "",
			SystemErr:  strings.Join(conflicts, "/n"),
			Totals:     junit.Totals{
				Tests: testStatus[junit.StatusPassed] + testStatus[junit.StatusSkipped] + testStatus[junit.StatusFailed] + testStatus[junit.StatusError],
				Passed: testStatus[junit.StatusPassed],
				Skipped: testStatus[junit.StatusSkipped],
				Failed: testStatus[junit.StatusFailed],
				Error: testStatus[junit.StatusError],
				Duration: totalExecutionTime,
			},
		},
	}, len(conflicts) > 0
}

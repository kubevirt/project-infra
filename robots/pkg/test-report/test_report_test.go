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

package test_report

import (
	"reflect"
	"regexp"
	"testing"
)

func Test_CreateReportData(t *testing.T) {
	type args struct {
		testNameFilterDefaultRegexp            *regexp.Regexp
		testNamesToJobNamesToExecutionStatus   map[string]map[string]int
		jobNamePatternsToTestNameFilterRegexps map[*regexp.Regexp]*regexp.Regexp
	}
	tests := []struct {
		name string
		args args
		want Data
	}{
		{
			name: "test has no data",
			args: args{
				testNameFilterDefaultRegexp: regexp.MustCompile("blah"),
				testNamesToJobNamesToExecutionStatus: map[string]map[string]int{
					"testName": {
						"jobName": TestExecution_NoData,
					},
				},
			},
			want: NewData(
				[]string{"testName"},                           // testNames
				map[string]interface{}{},                       // filteredTestNames
				map[string]interface{}{"testName": struct{}{}}, // skippedTests
				[]string{"jobName"},                            // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"jobName": TestExecution_NoData,
					},
				}, // testNamesToJobNamesToExecutionStatus
			),
		},
		{
			name: "test is run",
			args: args{
				testNameFilterDefaultRegexp: regexp.MustCompile("blah"),
				testNamesToJobNamesToExecutionStatus: map[string]map[string]int{
					"testName": {
						"jobName": TestExecution_Run,
					},
				},
			},
			want: NewData(
				[]string{"testName"},     // testNames
				map[string]interface{}{}, // filteredTestNames
				map[string]interface{}{}, // skippedTests
				[]string{"jobName"},      // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"jobName": TestExecution_Run,
					},
				}, // testNamesToJobNamesToExecutionStatus
			),
		},
		{
			name: "test is skipped",
			args: args{
				testNameFilterDefaultRegexp: regexp.MustCompile("blah"),
				testNamesToJobNamesToExecutionStatus: map[string]map[string]int{
					"testName": {
						"jobName": TestExecution_Skipped,
					},
				},
			},
			want: NewData(
				[]string{"testName"},                           // testNames
				map[string]interface{}{},                       // filteredTestNames
				map[string]interface{}{"testName": struct{}{}}, // skippedTests
				[]string{"jobName"},                            // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"jobName": TestExecution_Skipped,
					},
				}, // testNamesToJobNamesToExecutionStatus
			),
		},
		{
			name: "test is inside dont_run.json for one test lane and skipped in the other",
			args: args{
				testNameFilterDefaultRegexp: regexp.MustCompile("testNameDefault"),
				testNamesToJobNamesToExecutionStatus: map[string]map[string]int{
					"testName": {
						"test-version-1.2-lane": TestExecution_Skipped,
						"test-version-2.3-lane": TestExecution_Skipped,
					},
				},
				jobNamePatternsToTestNameFilterRegexps: map[*regexp.Regexp]*regexp.Regexp{
					regexp.MustCompile(`version-1\.2`): regexp.MustCompile("testName"),
					regexp.MustCompile(`version-2\.3`): regexp.MustCompile("testName42"),
				},
			},
			/*
				Outcome should be:
				* it DOES appear in the testcases
				* it doesn't appear in the filtered testcases
				* it DOES appear in the skipped testcases
				* rewrite of map to include the dont_run aka TestExecution_Unsupported
			*/
			want: NewData(
				[]string{"testName"},                                       // testNames
				map[string]interface{}{},                                   // filteredTestNames
				map[string]interface{}{"testName": struct{}{}},             // skippedTests
				[]string{"test-version-1.2-lane", "test-version-2.3-lane"}, // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"test-version-1.2-lane": TestExecution_Unsupported,
						"test-version-2.3-lane": TestExecution_Skipped,
					},
				}, // testNamesToJobNamesToExecutionStatus
			),
		},
		{
			name: "test is inside dont_run.json for one test lane and run in the other",
			args: args{
				testNameFilterDefaultRegexp: regexp.MustCompile("testNameDefault"),
				testNamesToJobNamesToExecutionStatus: map[string]map[string]int{
					"testName": {
						"test-version-1.2-lane": TestExecution_Skipped,
						"test-version-2.3-lane": TestExecution_Run,
					},
				},
				jobNamePatternsToTestNameFilterRegexps: map[*regexp.Regexp]*regexp.Regexp{
					regexp.MustCompile(`version-1\.2`): regexp.MustCompile("testName"),
					regexp.MustCompile(`version-2\.3`): regexp.MustCompile("testName42"),
				},
			},
			/*
				Outcome should be:
				* it DOES appear in the testcases
				* it doesn't appear in the filtered testcases
				* it doesn't appear in the skipped testcases
				* rewrite of map to include the dont_run aka TestExecution_Unsupported
			*/
			want: NewData(
				[]string{"testName"},     // testNames
				map[string]interface{}{}, // filteredTestNames
				map[string]interface{}{}, // skippedTests
				[]string{"test-version-1.2-lane", "test-version-2.3-lane"}, // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"test-version-1.2-lane": TestExecution_Unsupported,
						"test-version-2.3-lane": TestExecution_Run,
					},
				}, // testNamesToJobNamesToExecutionStatus
			),
		},
		{
			name: "test is inside dont_run.json for all test lanes",
			args: args{
				testNameFilterDefaultRegexp: regexp.MustCompile("testNameDefault"),
				testNamesToJobNamesToExecutionStatus: map[string]map[string]int{
					"testName": {
						"test-version-1.2-lane": TestExecution_Skipped,
						"test-version-2.3-lane": TestExecution_Skipped,
					},
				},
				jobNamePatternsToTestNameFilterRegexps: map[*regexp.Regexp]*regexp.Regexp{
					regexp.MustCompile(`version-1\.2`): regexp.MustCompile("testName"),
					regexp.MustCompile(`version-2\.3`): regexp.MustCompile("testName"),
				},
			},
			/*
				Outcome should be:
				* it does appear in the testcases
				* it does appear in the filtered testcases
				* it doesn't appear in the skipped testcases
				* rewrite of map to include the dont_run aka TestExecution_Unsupported for all
			*/
			want: NewData(
				[]string{"testName"},                                       // testNames
				map[string]interface{}{"testName": struct{}{}},             // filteredTestNames
				map[string]interface{}{"testName": struct{}{}},             // skippedTests
				[]string{"test-version-1.2-lane", "test-version-2.3-lane"}, // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"test-version-1.2-lane": TestExecution_Unsupported,
						"test-version-2.3-lane": TestExecution_Unsupported,
					},
				}, // testNamesToJobNamesToExecutionStatus
			),
		},
		{
			name: "test is inside dont_run.json for one test lane, run on another and skipped on a third",
			args: args{
				testNameFilterDefaultRegexp: regexp.MustCompile("testNameDefault"),
				testNamesToJobNamesToExecutionStatus: map[string]map[string]int{
					"testName": {
						"test-version-1.2-lane": TestExecution_Skipped,
						"test-version-2.3-lane": TestExecution_Skipped,
						"test-version-3.4-lane": TestExecution_Run,
					},
				},
				jobNamePatternsToTestNameFilterRegexps: map[*regexp.Regexp]*regexp.Regexp{
					regexp.MustCompile(`version-1\.2`): regexp.MustCompile("testName"),
					regexp.MustCompile(`version-2\.3`): regexp.MustCompile("testName42"),
					regexp.MustCompile(`version-3\.4`): regexp.MustCompile("testName42"),
				},
			},
			/*
				Outcome should be:
				* it appears in the testcases
				* it doesn't appear in the filtered testcases
				* it doesn't appear in the skipped testcases
				* rewrite of map to include the dont_run aka TestExecution_Unsupported
			*/
			want: NewData(
				[]string{"testName"},     // testNames
				map[string]interface{}{}, // filteredTestNames
				map[string]interface{}{}, // skippedTests
				[]string{"test-version-1.2-lane", "test-version-2.3-lane", "test-version-3.4-lane"}, // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"test-version-1.2-lane": TestExecution_Unsupported,
						"test-version-2.3-lane": TestExecution_Skipped,
						"test-version-3.4-lane": TestExecution_Run,
					},
				}, // testNamesToJobNamesToExecutionStatus
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CreateReportData(tt.args.jobNamePatternsToTestNameFilterRegexps, tt.args.testNamesToJobNamesToExecutionStatus); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateReportData() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

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

package main

import (
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"reflect"
	"regexp"
	"testing"
)

func Test_writeHTMLReportToOutput(t *testing.T) {
	type args struct {
		htmlReportOutputWriter       io.Writer
		testNames                    []string
		filteredTestNames            []string
		skippedTests                 map[string]interface{}
		lookedAtJobs                 []string
		testNamesToJobNamesToSkipped map[string]map[string]int
		err                          error
		jLog                         *logrus.Entry
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test template",
			args: args{
				htmlReportOutputWriter: os.Stdout,
				testNames:              []string{"a", "b", "c"},
				filteredTestNames:      []string{"la", "le", "lu"},
				skippedTests: map[string]interface{}{
					"a": struct{}{}},
				lookedAtJobs: []string{"job1", "job2", "job3"},
				testNamesToJobNamesToSkipped: map[string]map[string]int{
					"a": {
						"job1": test_execution_skipped,
						"job2": test_execution_skipped,
					},
					"b": {
						"job1": test_execution_skipped,
						"job2": test_execution_run,
						"job3": test_execution_run,
					},
					"c": {
						"job1": test_execution_skipped,
						"job2": test_execution_skipped,
						"job3": test_execution_run,
					},
				},
				err:  nil,
				jLog: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := writeHTMLReportToOutput(newData(tt.args.testNames, tt.args.filteredTestNames, tt.args.skippedTests, tt.args.lookedAtJobs, tt.args.testNamesToJobNamesToSkipped), tt.args.htmlReportOutputWriter)
			if err != nil {
				t.Errorf("writeHTMLReportToOutput: %v", err)
			}
		})
	}
}

func Test_createReportData(t *testing.T) {
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
						"jobName": test_execution_no_data,
					},
				},
			},
			want: newData(
				[]string{"testName"}, // testNames
				[]string{},           // filteredTestNames
				map[string]interface{}{"testName": struct{}{}}, // skippedTests
				[]string{"jobName"},                            // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"jobName": test_execution_no_data,
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
						"jobName": test_execution_run,
					},
				},
			},
			want: newData(
				[]string{"testName"},     // testNames
				[]string{},               // filteredTestNames
				map[string]interface{}{}, // skippedTests
				[]string{"jobName"},      // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"jobName": test_execution_run,
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
						"jobName": test_execution_skipped,
					},
				},
			},
			want: newData(
				[]string{"testName"}, // testNames
				[]string{},           // filteredTestNames
				map[string]interface{}{"testName": struct{}{}}, // skippedTests
				[]string{"jobName"},                            // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"jobName": test_execution_skipped,
					},
				}, // testNamesToJobNamesToExecutionStatus
			),
		},
		{
			name: "test is filtered by default regexp",
			args: args{
				testNameFilterDefaultRegexp: regexp.MustCompile("testName"),
				testNamesToJobNamesToExecutionStatus: map[string]map[string]int{
					"testName": {
						"jobName": test_execution_skipped,
					},
				},
			},
			want: newData(
				[]string{},               // testNames
				[]string{"testName"},     // filteredTestNames
				map[string]interface{}{}, // skippedTests
				[]string{},               // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"jobName": test_execution_skipped,
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
						"test-version-1.2-lane": test_execution_skipped,
						"test-version-2.3-lane": test_execution_skipped,
					},
				},
				jobNamePatternsToTestNameFilterRegexps: map[*regexp.Regexp]*regexp.Regexp{
					regexp.MustCompile("version-1\\.2"): regexp.MustCompile("testName"),
					regexp.MustCompile("version-2\\.3"): regexp.MustCompile("testName42"),
				},
			},
			/*
				Outcome should be:
				* it DOES appear in the testcases
				* it doesn't appear in the filtered testcases
				* it DOES appear in the skipped testcases
				* rewrite of map to include the dont_run aka test_execution_unsupported
			*/
			want: newData(
				[]string{"testName"}, // testNames
				[]string{},           // filteredTestNames
				map[string]interface{}{"testName": struct{}{}},             // skippedTests
				[]string{"test-version-1.2-lane", "test-version-2.3-lane"}, // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"test-version-1.2-lane": test_execution_unsupported,
						"test-version-2.3-lane": test_execution_skipped,
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
						"test-version-1.2-lane": test_execution_skipped,
						"test-version-2.3-lane": test_execution_run,
					},
				},
				jobNamePatternsToTestNameFilterRegexps: map[*regexp.Regexp]*regexp.Regexp{
					regexp.MustCompile("version-1\\.2"): regexp.MustCompile("testName"),
					regexp.MustCompile("version-2\\.3"): regexp.MustCompile("testName42"),
				},
			},
			/*
				Outcome should be:
				* it DOES appear in the testcases
				* it doesn't appear in the filtered testcases
				* it doesn't appear in the skipped testcases
				* rewrite of map to include the dont_run aka test_execution_unsupported
			*/
			want: newData(
				[]string{"testName"},     // testNames
				[]string{},               // filteredTestNames
				map[string]interface{}{}, // skippedTests
				[]string{"test-version-1.2-lane", "test-version-2.3-lane"}, // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"test-version-1.2-lane": test_execution_unsupported,
						"test-version-2.3-lane": test_execution_run,
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
						"test-version-1.2-lane": test_execution_skipped,
						"test-version-2.3-lane": test_execution_skipped,
					},
				},
				jobNamePatternsToTestNameFilterRegexps: map[*regexp.Regexp]*regexp.Regexp{
					regexp.MustCompile("version-1\\.2"): regexp.MustCompile("testName"),
					regexp.MustCompile("version-2\\.3"): regexp.MustCompile("testName"),
				},
			},
			/*
				Outcome should be:
				* it doesn't appear in the testcases
				* it does appear in the filtered testcases
				* it doesn't appear in the skipped testcases
				* rewrite of map to include the dont_run aka test_execution_unsupported for all
			*/
			want: newData(
				[]string{},           // testNames
				[]string{"testName"}, // filteredTestNames
				map[string]interface{}{"testName": struct{}{}},             // skippedTests
				[]string{"test-version-1.2-lane", "test-version-2.3-lane"}, // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"test-version-1.2-lane": test_execution_unsupported,
						"test-version-2.3-lane": test_execution_unsupported,
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
						"test-version-1.2-lane": test_execution_skipped,
						"test-version-2.3-lane": test_execution_skipped,
						"test-version-3.4-lane": test_execution_run,
					},
				},
				jobNamePatternsToTestNameFilterRegexps: map[*regexp.Regexp]*regexp.Regexp{
					regexp.MustCompile("version-1\\.2"): regexp.MustCompile("testName"),
					regexp.MustCompile("version-2\\.3"): regexp.MustCompile("testName42"),
					regexp.MustCompile("version-3\\.4"): regexp.MustCompile("testName42"),
				},
			},
			/*
				Outcome should be:
				* it appears in the testcases
				* it doesn't appear in the filtered testcases
				* it doesn't appear in the skipped testcases
				* rewrite of map to include the dont_run aka test_execution_unsupported
			*/
			want: newData(
				[]string{"testName"},     // testNames
				[]string{},               // filteredTestNames
				map[string]interface{}{}, // skippedTests
				[]string{"test-version-1.2-lane", "test-version-2.3-lane", "test-version-3.4-lane"}, // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"test-version-1.2-lane": test_execution_unsupported,
						"test-version-2.3-lane": test_execution_skipped,
						"test-version-3.4-lane": test_execution_run,
					},
				}, // testNamesToJobNamesToExecutionStatus
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createReportData(tt.args.testNameFilterDefaultRegexp, tt.args.jobNamePatternsToTestNameFilterRegexps, tt.args.testNamesToJobNamesToExecutionStatus); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createReportData() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

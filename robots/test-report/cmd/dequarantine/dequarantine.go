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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package dequarantine

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	kvjenkins "kubevirt.io/project-infra/pkg/jenkins"
	testreport "kubevirt.io/project-infra/pkg/test-report"
)

var dequarantineCmd = &cobra.Command{
	Use:   "dequarantine",
	Short: "has subcommands related to dequarantining tests",
	Run:   func(cmd *cobra.Command, args []string) {},
}

var logger = logrus.StandardLogger().WithField("test-report", "dequarantine")

type quarantinedTestsRunData struct {
	*testreport.FilterTestRecord
	testNamePattern *regexp.Regexp
	Tests           []*quarantinedTestRunsData `json:"tests"`
}

func (q *quarantinedTestsRunData) addIfMatchesTestName(data *quarantinedTestRunsData) bool {
	added := q.testNamePattern.MatchString(data.TestName)
	if added {
		q.Tests = append(q.Tests, data)
	}
	return added
}

type quarantinedTestRunsData struct {
	TestName    string                    `json:"testName"`
	TestResults []*quarantinedTestRunData `json:"testResults"`
}

func (q *quarantinedTestRunsData) sortTestResults() {
	sort.Slice(
		q.TestResults,
		func(i, j int) bool {
			return q.TestResults[i].BuildNo > q.TestResults[j].BuildNo
		},
	)
}

type quarantinedTestRunData struct {
	BuildNo  int64     `json:"buildNo"`
	DateTime time.Time `json:"dateTime"`
	Result   string    `json:"result"`
}

func DequarantineCmd(rootLogger *logrus.Entry) *cobra.Command {
	logger = rootLogger
	return dequarantineCmd
}

func init() {
	dequarantineCmd.AddCommand(dequarantineReportCmd)
	dequarantineCmd.AddCommand(dequarantineExecuteCmd)
}

func generateDequarantineBaseData(jenkins *gojenkins.Jenkins, ctx context.Context, jobs []*gojenkins.Job, startOfReport time.Time, quarantinedTestEntriesFromFile []*testreport.FilterTestRecord) []*quarantinedTestsRunData {

	quarantinedTestsRunDataValues, testNamePattern := createTestCaseCollectionBaseData(quarantinedTestEntriesFromFile)
	quarantinedTestNamesToTestRunData := filterMatchingTestRunData(jobs, startOfReport, ctx, jenkins, testNamePattern)
	insertTestRunDataIntoTestCases(quarantinedTestNamesToTestRunData, quarantinedTestsRunDataValues)
	return quarantinedTestsRunDataValues
}

// createTestCaseCollectionBaseData returns the base records that are used to collect the test runs for those tests
// and the regexp.Regexp that is used to filter the test data by test name that matches the quarantined file expressions
func createTestCaseCollectionBaseData(quarantinedTestEntriesFromFile []*testreport.FilterTestRecord) ([]*quarantinedTestsRunData, *regexp.Regexp) {
	var quarantinedTestPatternStrings []string
	var quarantinedTestsRunDataValues []*quarantinedTestsRunData
	for _, quarantinedTestEntry := range quarantinedTestEntriesFromFile {
		quarantinedTestsRunDataValues = append(quarantinedTestsRunDataValues, &quarantinedTestsRunData{
			FilterTestRecord: quarantinedTestEntry,
			testNamePattern:  regexp.MustCompile(regexp.QuoteMeta(quarantinedTestEntry.Id)),
		})
		quarantinedTestPatternStrings = append(quarantinedTestPatternStrings, regexp.QuoteMeta(quarantinedTestEntry.Id))
	}
	testNamePattern := regexp.MustCompile(strings.Join(quarantinedTestPatternStrings, "|"))
	return quarantinedTestsRunDataValues, testNamePattern
}

// filterMatchingTestRunData fetches all test cases that match any entry from the quarantined file, and returns a map
// using test name and array of test cases
func filterMatchingTestRunData(jobs []*gojenkins.Job, startOfReport time.Time, ctx context.Context, jenkins *gojenkins.Jenkins, testNamePattern *regexp.Regexp) map[string]*quarantinedTestRunsData {
	testNamesToTestCases := map[string]*quarantinedTestRunsData{}
	for _, job := range jobs {
		buildNumbersToTestResultsForJob := kvjenkins.GetBuildNumbersToTestResultsForJob(startOfReport, job, ctx, logger)
		for buildNumber, testResultForJob := range buildNumbersToTestResultsForJob {
			build, err := jenkins.GetBuild(ctx, job.GetName(), buildNumber)
			if err != nil {
				logger.Fatalf("failed to get build data of build %d for job %q: %v", buildNumber, job.GetName(), err)
			}
			for _, suite := range testResultForJob.Suites {
				for _, testCase := range suite.Cases {
					if !testNamePattern.MatchString(testCase.Name) {
						continue
					}
					if _, exists := testNamesToTestCases[testCase.Name]; !exists {
						testNamesToTestCases[testCase.Name] = &quarantinedTestRunsData{
							TestName:    testCase.Name,
							TestResults: []*quarantinedTestRunData{},
						}
					}
					quarantinedTestRunDataEntry := &quarantinedTestRunData{
						BuildNo:  buildNumber,
						DateTime: build.GetTimestamp(),
						Result:   testCase.Status,
					}
					testNamesToTestCases[testCase.Name].TestResults = append(testNamesToTestCases[testCase.Name].TestResults, quarantinedTestRunDataEntry)
				}
			}
		}
	}
	return testNamesToTestCases
}

// insertTestRunDataIntoTestCases sorts every set of test case below it's matching test id
func insertTestRunDataIntoTestCases(testNamesToTestCases map[string]*quarantinedTestRunsData, quarantinedTestsRunDataValues []*quarantinedTestsRunData) {
	for _, testCases := range testNamesToTestCases {
		testCases.sortTestResults()
		for _, quarantinedTestsRunDataValue := range quarantinedTestsRunDataValues {
			if quarantinedTestsRunDataValue.addIfMatchesTestName(testCases) {
				break
			}
		}
	}
}

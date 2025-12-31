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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/bndr/gojenkins"
	log "github.com/sirupsen/logrus"
	"kubevirt.io/project-infra/pkg/jenkins"
)

const (
	TestExecution_NoData = iota
	TestExecution_Skipped
	TestExecution_Run
	TestExecution_Unsupported
)

const (
	DefaultJenkinsBaseUrl = "https://main-jenkins-csb-cnvqe.apps.ocp-c1.prod.psi.redhat.com/"
)

var logger *log.Entry

func init() {
	logger = log.StandardLogger().WithField("robot", "test-report")
}

type Data struct {

	// JenkinsBaseURL is the URL pointing to the Jenkins instance to query for build data
	JenkinsBaseURL string `json:"jenkinsBaseURL"`

	// TestNames contains the names of all tests that have not been filtered on all lanes
	TestNames []string `json:"testNames"`

	// FilteredTestNames contains the names of all tests that have been filtered on all lanes
	FilteredTestNames map[string]interface{} `json:"filteredTestNames"`

	// SkippedTests contains the test names for all tests that have been skipped on all lanes, aka not having been run on any lane
	SkippedTests map[string]interface{} `json:"skippedTests"`

	// LookedAtJobs contains the names of all test lanes that have been looked at
	LookedAtJobs []string `json:"lookedAtJobs"`

	// TestNamesToJobNamesToSkipped contains a map of test names per test pointing to the jobs where that test has been seen, which
	// points to the state that was seen on that lane.
	// See TestExecution_NoData, TestExecution_Skipped, TestExecution_Run, TestExecution_Unsupported
	TestNamesToJobNamesToSkipped map[string]map[string]int `json:"testNamesToJobNamesToSkipped"`

	// TestExecutionMapping is a map containing the string names of the const that is used inside the template
	TestExecutionMapping map[string]int

	// StartOfReport contains the lower end of the data interval for the report
	StartOfReport string

	// EndOfReport contains the upper end of the data interval for the report
	EndOfReport string

	// ReportConfig holds the full report configuration for displaying it inside the report
	ReportConfig string

	// ReportConfigName holds the name of the report configuration being used for this report
	ReportConfigName string
}

func (d Data) String() string {
	return fmt.Sprintf(`{
	JenkinsBaseURL: %s,
	TestNames: %v,
	FilteredTestNames: %v,
	SkippedTests: %v,
	LookedAtJobs: %v,
	TestNamesToJobNamesToSkipped: %v,
	TestExecutionMapping: %v,
}`, d.JenkinsBaseURL, d.TestNames, d.FilteredTestNames, d.SkippedTests, d.LookedAtJobs, d.TestNamesToJobNamesToSkipped, d.TestExecutionMapping)
}

func (d *Data) SetDataRange(startOfReport, endOfReport time.Time) {
	d.StartOfReport, d.EndOfReport = startOfReport.Format(time.RFC1123), endOfReport.Format(time.RFC1123)
}

func (d *Data) SetReportConfig(reportConfig string) {
	d.ReportConfig = reportConfig
}

func (d *Data) SetReportConfigName(name string) {
	d.ReportConfigName = name
}

func NewData(testNames []string, filteredTestNames map[string]interface{}, skippedTests map[string]interface{}, lookedAtJobs []string, testNamesToJobNamesToSkipped map[string]map[string]int) Data {
	return Data{
		TestNames:                    testNames,
		FilteredTestNames:            filteredTestNames,
		SkippedTests:                 skippedTests,
		LookedAtJobs:                 lookedAtJobs,
		TestNamesToJobNamesToSkipped: testNamesToJobNamesToSkipped,
		JenkinsBaseURL:               DefaultJenkinsBaseUrl,
		TestExecutionMapping: map[string]int{
			"TestExecution_NoData":      TestExecution_NoData,
			"TestExecution_Skipped":     TestExecution_Skipped,
			"TestExecution_Run":         TestExecution_Run,
			"TestExecution_Unsupported": TestExecution_Unsupported,
		},
	}
}

// Config is the configuration for the report
type Config struct {

	// JobNamePattern is a regexp.Regexp that describes which jobs are considered for the report
	JobNamePattern string `yaml:"jobNamePattern"`

	// JobNamePatternForTestNames is a regexp.Regexp that describes which jobs are considered to contain all test names
	// we should be looking at
	JobNamePatternForTestNames string `yaml:"jobNamePatternForTestNames,omitempty"`

	// TestNamePattern is a regexp.Regexp that describes what tests are considered for the report
	TestNamePattern string `yaml:"testNamePattern"`

	// JobNamePatternsToDontRunFileURLs is an array where each entry describes which tests are filtered regarding the
	// `dont_run_tests.json` if the pattern matches the job name
	JobNamePatternsToDontRunFileURLs []*JobNamePatternToDontRunFileURL `yaml:"jobNamePatternsToDontRunFileURLs"`

	// MaxConnsPerHost sets a boundary to the maximum number of parallel connections to the Jenkins
	MaxConnsPerHost int `yaml:"maxConnsPerHost"`
}

type JobNamePatternToDontRunFileURL struct {

	// JobNamePattern describes what jobs match to a `dont_run_tests.json` file in order to filter out those tests
	JobNamePattern string `yaml:"jobNamePattern"`

	// DontRunFileURL is the URL to a `dont_run_tests.json` file
	DontRunFileURL string `yaml:"dontRunFileURL"`
}

type FilterTestRecord struct {
	Id     string `json:"id"`
	Reason string `json:"reason"`
}

func (r *FilterTestRecord) String() string {
	return fmt.Sprintf("{id: %q, reason: %q}", r.Id, r.Reason)
}

func CreateReportData(jobNamePatternsToTestNameFilterRegexps map[*regexp.Regexp]*regexp.Regexp, testNamesToJobNamesToExecutionStatus map[string]map[string]int) Data {
	testNames := []string{}
	skippedTests := map[string]interface{}{}
	filteredTestNames := map[string]interface{}{}
	lookedAtJobsMap := map[string]interface{}{}

	for testName, jobNamesToSkipped := range testNamesToJobNamesToExecutionStatus {
		testSkipped := true
		filteredOnAllLanes := true
		for jobName, executionStatus := range jobNamesToSkipped {
			if _, exists := lookedAtJobsMap[jobName]; !exists {
				lookedAtJobsMap[jobName] = struct{}{}
			}
			switch executionStatus {
			case TestExecution_Run:
				testSkipped = false
				filteredOnAllLanes = false
			case TestExecution_Skipped:
				jobNameMatcherFound := false
				for jobNameMatcher, testNameMatcher := range jobNamePatternsToTestNameFilterRegexps {
					if jobNameMatcher.MatchString(jobName) {
						if testNameMatcher.MatchString(testName) {
							testNamesToJobNamesToExecutionStatus[testName][jobName] = TestExecution_Unsupported
						} else {
							filteredOnAllLanes = false
						}
						jobNameMatcherFound = true
						break
					}
				}
				if !jobNameMatcherFound {
					filteredOnAllLanes = false
				}
			case TestExecution_NoData:
				filteredOnAllLanes = false
			}
		}
		if filteredOnAllLanes {
			filteredTestNames[testName] = struct{}{}
		}
		testNames = append(testNames, testName)
		if testSkipped {
			skippedTests[testName] = struct{}{}
		}
	}
	lookedAtJobs := []string{}
	for jobName := range lookedAtJobsMap {
		lookedAtJobs = append(lookedAtJobs, jobName)
	}

	sort.Strings(testNames)
	sort.Strings(lookedAtJobs)
	data := NewData(testNames, filteredTestNames, skippedTests, lookedAtJobs, testNamesToJobNamesToExecutionStatus)
	return data
}

func GetTestNamesToJobNamesToTestExecutions(jobs []*gojenkins.Job, startOfReport time.Time, ctx context.Context, testNamePattern *regexp.Regexp, jobNamePatternForTestNames *regexp.Regexp) map[string]map[string]int {
	resultsChan := make(chan map[string]map[string]int)
	go getTestNamesToJobNamesToTestExecutionForAllJobs(resultsChan, jobs, startOfReport, ctx, logger)

	testNamesToJobNamesToExecutionStatus := map[string]map[string]int{}

	for result := range resultsChan {
		for testName, jobNamesToExecutionStatus := range result {
			if !testNamePattern.MatchString(testName) {
				continue
			}
			if _, exists := testNamesToJobNamesToExecutionStatus[testName]; exists {
				for jobName, executionStatus := range jobNamesToExecutionStatus {
					testNamesToJobNamesToExecutionStatus[testName][jobName] = executionStatus
				}
			} else {
				testNamesToJobNamesToExecutionStatus[testName] = jobNamesToExecutionStatus
			}
		}
	}

	if jobNamePatternForTestNames != nil {
		for testName, jobNamesToExecutionStatus := range testNamesToJobNamesToExecutionStatus {
			jobNamePatternMatchesAnyJobNameForTestNames := false
			for jobName := range jobNamesToExecutionStatus {
				if jobNamePatternForTestNames.MatchString(jobName) {
					jobNamePatternMatchesAnyJobNameForTestNames = true
					break
				}
			}
			if !jobNamePatternMatchesAnyJobNameForTestNames {
				delete(testNamesToJobNamesToExecutionStatus, testName)
			}
		}
	}

	return testNamesToJobNamesToExecutionStatus
}

func getTestNamesToJobNamesToTestExecutionForAllJobs(resultsChan chan map[string]map[string]int, jobs []*gojenkins.Job, startOfReport time.Time, ctx context.Context, jLog *log.Entry) {

	var wg sync.WaitGroup
	wg.Add(len(jobs))

	defer close(resultsChan)
	for _, job := range jobs {
		fLog := jLog.WithField("job", job.GetName())
		go getTestNamesToJobNamesToTestExecutionForJob(startOfReport, ctx, fLog, job, resultsChan, &wg)
	}

	wg.Wait()
	jLog.Printf("done get all jobs")
}

func getTestNamesToJobNamesToTestExecutionForJob(startOfReport time.Time, ctx context.Context, jLog *log.Entry, job *gojenkins.Job, resultsChan chan map[string]map[string]int, wg *sync.WaitGroup) {
	defer wg.Done()
	testResultsForJob := jenkins.GetBuildNumbersToTestResultsForJob(startOfReport, job, ctx, jLog)
	testNamesToJobNamesToSkippedForJobName := map[string]map[string]int{}
	for _, testResultForJob := range testResultsForJob {
		for _, suite := range testResultForJob.Suites {
			for _, suiteCase := range suite.Cases {
				if _, exists := testNamesToJobNamesToSkippedForJobName[suiteCase.Name]; !exists {
					testNamesToJobNamesToSkippedForJobName[suiteCase.Name] = map[string]int{}
				}
				if suiteCase.Skipped {
					testNamesToJobNamesToSkippedForJobName[suiteCase.Name][job.GetName()] = TestExecution_Skipped
				} else {
					testNamesToJobNamesToSkippedForJobName[suiteCase.Name][job.GetName()] = TestExecution_Run
				}
			}
		}
	}
	resultsChan <- testNamesToJobNamesToSkippedForJobName
}

func CreateJobNamePatternsToTestNameFilterRegexps(config *Config, client *http.Client) (map[*regexp.Regexp]*regexp.Regexp, error) {

	jobNamePatternsToTestNameFilterRegexpsResult := map[*regexp.Regexp]*regexp.Regexp{}
	for _, jobNamePatternToDontRunFileURL := range config.JobNamePatternsToDontRunFileURLs {
		jobNamePattern := regexp.MustCompile(jobNamePatternToDontRunFileURL.JobNamePattern)
		dontRunFileURL := jobNamePatternToDontRunFileURL.DontRunFileURL
		completeFilterRegex, err := createFilterRegexFromDontRunFileEntries(dontRunFileURL, client)
		if err != nil {
			return nil, err
		}
		logger.Infof("for jobNamePattern %q filter expression is %q", jobNamePattern, completeFilterRegex)
		jobNamePatternsToTestNameFilterRegexpsResult[jobNamePattern] = completeFilterRegex
	}
	return jobNamePatternsToTestNameFilterRegexpsResult, nil
}

func createFilterRegexFromDontRunFileEntries(dontRunFileURL string, client *http.Client) (*regexp.Regexp, error) {
	filterTestRecords, err := FetchDontRunEntriesFromFile(dontRunFileURL, client)
	if err != nil {
		return nil, err
	}
	var testNameFilterRegexps []string
	for _, record := range filterTestRecords {
		testNameFilterRegexps = append(testNameFilterRegexps, regexp.QuoteMeta(record.Id))
	}
	return regexp.MustCompile(strings.Join(testNameFilterRegexps, "|")), nil
}

func FetchDontRunEntriesFromFile(dontRunFileURL string, client *http.Client) ([]*FilterTestRecord, error) {
	logger.Infof("fetching filter file %q", dontRunFileURL)
	response, err := client.Get(dontRunFileURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %q: %v", dontRunFileURL, err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("when fetching %q received status code: %d", dontRunFileURL, response.StatusCode)
	}

	defer response.Body.Close()
	var records []*FilterTestRecord
	err = json.NewDecoder(response.Body).Decode(&records)
	if err != nil {
		return nil, fmt.Errorf("failed to decode %q: %v", dontRunFileURL, err)
	}
	return records, nil
}

func FilterMatchingJobsByJobNamePattern(ctx context.Context, jenkins *gojenkins.Jenkins, innerJobs []gojenkins.InnerJob, jobNamePattern *regexp.Regexp) ([]*gojenkins.Job, error) {
	filteredJobs := []*gojenkins.Job{}
	logger.Printf("Filtering for jobs matching %s", jobNamePattern)
	for _, innerJob := range innerJobs {
		if !jobNamePattern.MatchString(innerJob.Name) {
			continue
		}
		job, err := jenkins.GetJob(ctx, innerJob.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to get job %s: %v", innerJob.Name, err)
		}
		filteredJobs = append(filteredJobs, job)
	}
	logger.Printf("%d jobs left after filtering", len(filteredJobs))
	return filteredJobs, nil
}

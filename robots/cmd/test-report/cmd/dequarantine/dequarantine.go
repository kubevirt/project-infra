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

package dequarantine

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/bndr/gojenkins"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	robots_jenkins "kubevirt.io/project-infra/robots/pkg/jenkins"
	test_report "kubevirt.io/project-infra/robots/pkg/test-report"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

var dequarantineCmd = &cobra.Command{
	Use:   "dequarantine",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDequarantineReport()
	},
}

type dequarantineReportOpts struct {
	quarantineFileURL string
	endpoint          string
	startFrom         time.Duration
	jobNamePattern    string
	maxConnsPerHost   int
	dryRun            bool
	outputFile        string
}

var jobNamePattern *regexp.Regexp

var dequarantineReportOptions = dequarantineReportOpts{}

func (r *dequarantineReportOpts) Validate() error {
	if r.quarantineFileURL == "" {
		return fmt.Errorf("quarantineFileURL must be set")
	}
	if r.jobNamePattern == "" {
		return fmt.Errorf("jobNamePattern must be set")
	}
	_, err := regexp.Compile(r.jobNamePattern)
	if err != nil {
		return fmt.Errorf("jobNamePattern %q is not a valid regexp", r.jobNamePattern)
	}
	return nil
}

func init() {
	dequarantineCmd.PersistentFlags().StringVar(&dequarantineReportOptions.endpoint, "endpoint", test_report.DefaultJenkinsBaseUrl, "jenkins base url")
	dequarantineCmd.PersistentFlags().DurationVar(&dequarantineReportOptions.startFrom, "start-from", 10*24*time.Hour, "time period for report")
	dequarantineCmd.PersistentFlags().StringVar(&dequarantineReportOptions.quarantineFileURL, "quarantine-file-url", "", "the url to the quarantine file")
	dequarantineCmd.PersistentFlags().StringVar(&dequarantineReportOptions.jobNamePattern, "job-name-pattern", "", "the pattern to which all jobs have to match")
	dequarantineCmd.PersistentFlags().IntVar(&dequarantineReportOptions.maxConnsPerHost, "max-conns-per-host", 3, "the maximum number of connections that are going to be made")
	dequarantineCmd.PersistentFlags().StringVar(&dequarantineReportOptions.outputFile, "outputFile", "", "Path to output file, if not given, a temporary file will be used")
	dequarantineCmd.PersistentFlags().BoolVar(&dequarantineReportOptions.dryRun, "dry-run", true, "whether to only check what jobs are being considered and then exit")
}

var logger *logrus.Entry

func DequarantineCmd(rootLogger *logrus.Entry) *cobra.Command {
	logger = rootLogger
	return dequarantineCmd
}

func runDequarantineReport() error {

	err := dequarantineReportOptions.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate command line arguments: %v", err)
	}

	jobNamePattern = regexp.MustCompile(dequarantineReportOptions.jobNamePattern)

	client := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost: dequarantineReportOptions.maxConnsPerHost,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	ctx := context.Background()

	logger.Printf("Creating client for %s", dequarantineReportOptions.endpoint)
	jenkins := gojenkins.CreateJenkins(client, dequarantineReportOptions.endpoint)
	_, err = jenkins.Init(ctx)
	if err != nil {
		logger.Fatalf("failed to contact jenkins %s: %v", dequarantineReportOptions.endpoint, err)
	}

	jobNames, err := jenkins.GetAllJobNames(ctx)
	if err != nil {
		logger.Fatalf("failed to get jobs: %v", err)
	}
	jobs, err := test_report.FilterMatchingJobsByJobNamePattern(ctx, jenkins, jobNames, jobNamePattern)
	if err != nil {
		logger.Fatalf("failed to filter matching jobs: %v", err)
	}
	var filteredJobNames []string
	for _, job := range jobs {
		filteredJobNames = append(filteredJobNames, job.GetName())
	}
	logger.Infof("jobs that are being considered: %s", strings.Join(filteredJobNames, ", "))
	if dequarantineReportOptions.dryRun {
		logger.Warn("dry-run mode, exiting")
		return nil
	}
	if len(jobs) == 0 {
		logger.Warn("no jobs left, nothing to do")
		return nil
	}

	quarantinedTestEntriesFromFile, err := test_report.FetchDontRunEntriesFromFile(dequarantineReportOptions.quarantineFileURL, client)
	if err != nil {
		logger.Fatalf("failed to filter matching jobs: %v", err)
	}

	// #1: create the base data that collects the test cases
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

	startOfReport := time.Now().Add(-1 * dequarantineReportOptions.startFrom)

	// #2: fetch all test cases that match any entry, and put them into a map using test name and array of test cases
	testNamesToTestCases := map[string]*quarantinedTestRunsData{}
	for _, job := range jobs {
		buildNumbersToTestResultsForJob := robots_jenkins.GetBuildNumbersToTestResultsForJob(startOfReport, job, ctx, logger)
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

	// #3: sort every set of test case below it's matching test id
	for _, testCases := range testNamesToTestCases {
		testCases.sortTestResults()
		for _, quarantinedTestsRunDataValue := range quarantinedTestsRunDataValues {
			if quarantinedTestsRunDataValue.addIfMatchesTestName(testCases) {
				break
			}
		}
	}

	// #4: write report data
	var outputFile *os.File
	if dequarantineReportOptions.outputFile == "" {
		outputFile, err = os.CreateTemp("", "quarantined-tests-run-*.json")
		if err != nil {
			return fmt.Errorf("failed to write report: %v", err)
		}
		dequarantineReportOptions.outputFile = outputFile.Name()
	} else {
		outputFile, err = os.Create(dequarantineReportOptions.outputFile)
		if err != nil {
			return fmt.Errorf("failed to write report: %v", err)
		}
	}
	err = json.NewEncoder(outputFile).Encode(quarantinedTestsRunDataValues)
	if err != nil {
		return fmt.Errorf("failed to write report: %v", err)
	}
	logger.Infof("Report data written to %q", dequarantineReportOptions.outputFile)
	return nil
}

type quarantinedTestsRunData struct {
	*test_report.FilterTestRecord
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

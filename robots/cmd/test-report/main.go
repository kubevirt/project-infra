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
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bndr/gojenkins"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"kubevirt.io/project-infra/robots/pkg/flakefinder"
	flakejenkins "kubevirt.io/project-infra/robots/pkg/jenkins"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultJenkinsBaseUrl = "https://main-jenkins-csb-cnvqe.apps.ocp-c1.prod.psi.redhat.com/"
	defaultFilterFileUrl  = "https://gitlab.cee.redhat.com/contra/cnv-qe-automation/-/raw/master/tests/tier1/kubevirt/dont_run_tests.json"
	reportTemplate        = `
<html>
<head>
    <title>flakefinder report</title>
    <meta charset="UTF-8">
		<style>
			table, th, td {
				border: 1px solid black;
			}
			.yellow {
				background-color: #ffff80;
			}
			.almostgreen {
				background-color: #dfff80;
			}
			.green {
				background-color: #9fff80;
			}
			.red {
				background-color: #ff8080;
			}
			.orange {
				background-color: #ffbf80;
			}
			.unimportant {
			}
			.tests_passed {
				color: #226c18;
				font-weight: bold;
			}
			.tests_failed {
				color: #8a1717;
				font-weight: bold;
			}
			.tests_skipped {
				color: #535453;
				font-weight: bold;
			}
		</style>
	</meta>
</head>
<body>
<h1>test execution report</h1>

<table>
    <tr>
        <td></td>
        <td></td>
        {{ range $job := $.LookedAtJobs }}
        <td><a href="{{ $.JenkinsBaseURL }}/job/{{ $job }}/">{{ $job }}</a></td>
        {{ end }}
    </tr>
    {{ range $row, $test := $.TestNames }}
    <tr>
        <td><div id="row{{$row}}"><a href="#row{{$row}}">{{ $row }}</a></div></td>
        <td class="{{ if (index $.SkippedTests $test) }}red{{ end }}">{{ $test }}</td>
        {{ range $col, $job := $.LookedAtJobs }}
        <td class="center">{{ with $skipped := (index $.TestNamesToJobNamesToSkipped $test $job) }}
            <div id="r{{$row}}c{{$col}}" class="{{ if eq $skipped (index $.TestExecutionMapping "test_execution_skipped") }}yellow{{ else if eq $skipped (index $.TestExecutionMapping "test_execution_run") }}green{{ else }}{{ end }}" >
                <input title="{{ $test }} {{ $job }}" type="checkbox" readonly {{ if eq $skipped (index $.TestExecutionMapping "test_execution_run") }}checked{{ end }}/>
			</div>
		{{ else }}n/a{{ end }}</td>
        {{ end }}
    </tr>
    {{ end }}
</table>

</body>
</html>
`
)

var (
	rootCmd *cobra.Command
	opts    options
)

func init() {
	rootCmd = &cobra.Command{
		Use:   "test-report",
		Short: "test-report creates a report about which tests have been run on what lane",
		RunE:  runReport,
	}
	opts = options{}
	flagOptions()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}

type options struct {
	endpoint            string
	startFrom           time.Duration
	jobNamePattern      string
	outputFile          string
	overwrite           bool
	quarantinedFileUrls []string
}

func flagOptions() options {
	rootCmd.PersistentFlags().StringVar(&opts.endpoint, "endpoint", defaultJenkinsBaseUrl, "jenkins base url")
	rootCmd.PersistentFlags().DurationVar(&opts.startFrom, "start-from", 14*24*time.Hour, "time period for report")
	rootCmd.PersistentFlags().StringVar(&opts.jobNamePattern, "job-name-pattern", "^test-(ssp-cnv-4\\.[0-9]+|kubevirt-cnv-4\\.[0-9]+-(compute|network|operator)-[a-z]+)$", "jenkins job name go regex pattern to filter jobs for the report")
	rootCmd.PersistentFlags().StringVar(&opts.outputFile, "outputFile", "", "Path to output file, if not given, a temporary file will be used")
	rootCmd.PersistentFlags().BoolVar(&opts.overwrite, "overwrite", true, "overwrite output file")
	rootCmd.PersistentFlags().StringArrayVar(&opts.quarantinedFileUrls, "filter-file-urls", []string{defaultFilterFileUrl}, "file urls to use as filters for test cases, use quarantined_tests.json and/or dont_run_tests.json")
	return opts
}

func (o *options) Validate() error {
	if o.outputFile == "" {
		outputFile, err := os.CreateTemp("", "test-report-*.html")
		if err != nil {
			return fmt.Errorf("failed to write report: %v", err)
		}
		o.outputFile = outputFile.Name()
	} else {
		stat, err := os.Stat(o.outputFile)
		if err != nil && errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if stat.IsDir() {
			return fmt.Errorf("failed to write report, file %s is a directory", o.outputFile)
		}
		if err == nil && !o.overwrite {
			return fmt.Errorf("failed to write report, file %s exists", o.outputFile)
		}
	}
	return nil
}

const (
	test_execution_no_data = iota
	test_execution_skipped = iota
	test_execution_run     = iota
)

type FilterTestRecord struct {
	Id     string `json:"id"`
	Reason string `json:"reason"`
}

func runReport(cmd *cobra.Command, args []string) error {
	log.StandardLogger().SetFormatter(&log.JSONFormatter{})
	jLog := log.StandardLogger().WithField("robot", "test-report")

	err := opts.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate command line arguments: %v", err)
	}

	ctx := context.Background()

	client := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost: 5,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	jLog.Printf("Creating client for %s", opts.endpoint)
	jenkins := gojenkins.CreateJenkins(client, opts.endpoint)
	_, err = jenkins.Init(ctx)
	if err != nil {
		jLog.Fatalf("failed to contact jenkins %s: %v", opts.endpoint, err)
	}

	jobNames, err := jenkins.GetAllJobNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to get jobs: %v", err)
	}
	jobs := filterMatchingJobs(ctx, jenkins, jLog, opts, jobNames)
	if len(jobs) == 0 {
		jLog.Warn("no jobs left, nothing to do")
		return nil
	}

	startOfReport := time.Now().Add(-1 * opts.startFrom)

	resultsChan := make(chan map[string]map[string]bool)
	go getTestNamesToJobNamesToSkippedForAllJobs(resultsChan, jobs, startOfReport, ctx, jLog)

	testNamesToJobNamesToExecutionStatus := map[string]map[string]int{}
	for result := range resultsChan {
		for testName, jobNamesToSkipped := range result {
			if _, exists := testNamesToJobNamesToExecutionStatus[testName]; !exists {
				testNamesToJobNamesToExecutionStatus[testName] = map[string]int{}
			}
			for jobName, skipped := range jobNamesToSkipped {
				if skipped {
					testNamesToJobNamesToExecutionStatus[testName][jobName] = test_execution_skipped
				} else {
					testNamesToJobNamesToExecutionStatus[testName][jobName] = test_execution_run
				}
			}
		}
	}

	bytes, err := json.MarshalIndent(testNamesToJobNamesToExecutionStatus, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshall result: %v", err)
	}

	jsonFileName := strings.TrimSuffix(opts.outputFile, ".html") + ".json"
	jLog.Printf("Writing json to %q", jsonFileName)
	jsonOutputWriter, err := os.OpenFile(jsonFileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil && err != os.ErrNotExist {
		return fmt.Errorf("failed to write report: %v", err)
	}
	defer jsonOutputWriter.Close()

	jsonOutputWriter.Write(bytes)

	filterRegexes := []string{}
	for _, quarantinedFileUrl := range opts.quarantinedFileUrls {
		jLog.Infof("fetching filter file '%s'", quarantinedFileUrl)
		response, err := http.Get(quarantinedFileUrl)
		if err != nil {
			return fmt.Errorf("failed to fetch %s: %v", quarantinedFileUrl, err)
		}
		if response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusBadRequest {
			records := []*FilterTestRecord{}
			err := json.NewDecoder(response.Body).Decode(&records)
			if err != nil {
				return fmt.Errorf("failed to decode %s: %v", quarantinedFileUrl, err)
			}
			for _, record := range records {
				filterRegexes = append(filterRegexes, record.Id)
			}
		} else {
			return fmt.Errorf("when fetching %s received status code: %d", quarantinedFileUrl, response.StatusCode)
		}
	}
	completeFilterRegex := regexp.MustCompile(strings.Join(filterRegexes, "|"))
	jLog.Infof("filter expression is '%s'", completeFilterRegex)

	testNames := []string{}
	skippedTests := map[string]interface{}{}
	lookedAtJobsMap := map[string]interface{}{}

	for testName, jobNamesToSkipped := range testNamesToJobNamesToExecutionStatus {
		if completeFilterRegex.MatchString(testName) {
			jLog.Warnf("filtering %s", testName)
			continue
		}
		testNames = append(testNames, testName)
		testSkipped := true
		for jobName, executionStatus := range jobNamesToSkipped {
			if _, exists := lookedAtJobsMap[jobName]; !exists {
				lookedAtJobsMap[jobName] = struct{}{}
			}
			if executionStatus == test_execution_run {
				testSkipped = false
				break
			}
		}
		if testSkipped {
			skippedTests[testName] = struct{}{}
		}
	}
	lookedAtJobs := []string{}
	for jobName, _ := range lookedAtJobsMap {
		lookedAtJobs = append(lookedAtJobs, jobName)
	}

	sort.Strings(testNames)
	sort.Strings(lookedAtJobs)

	htmlReportOutputWriter, err := os.OpenFile(opts.outputFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to write report: %v", err)
	}
	defer htmlReportOutputWriter.Close()
	jLog.Printf("Writing html to %q", opts.outputFile)
	writeHTMLReportToOutput(htmlReportOutputWriter, testNames, skippedTests, lookedAtJobs, testNamesToJobNamesToExecutionStatus, err, jLog)
	return nil
}

func writeHTMLReportToOutput(htmlReportOutputWriter io.Writer, testNames []string, skippedTests map[string]interface{}, lookedAtJobs []string, testNamesToJobNamesToSkipped map[string]map[string]int, err error, jLog *log.Entry) {

	data := Data{
		TestNames:                    testNames,
		SkippedTests:                 skippedTests,
		LookedAtJobs:                 lookedAtJobs,
		TestNamesToJobNamesToSkipped: testNamesToJobNamesToSkipped,
		JenkinsBaseURL:               defaultJenkinsBaseUrl,
		TestExecutionMapping: map[string]int{
			"test_execution_no_data": test_execution_no_data,
			"test_execution_skipped": test_execution_skipped,
			"test_execution_run":     test_execution_run,
		},
	}
	err = flakefinder.WriteTemplateToOutput(reportTemplate, data, htmlReportOutputWriter)
	if err != nil {
		jLog.Fatalf("failed to write report: %v", err)
	}
}

type Data struct {
	JenkinsBaseURL               string                    `json:"jenkinsBaseURL"`
	TestNames                    []string                  `json:"testNames"`
	SkippedTests                 map[string]interface{}    `json:"skippedTests"`
	LookedAtJobs                 []string                  `json:"lookedAtJobs"`
	TestNamesToJobNamesToSkipped map[string]map[string]int `json:"testNamesToJobNamesToSkipped"`
	TestExecutionMapping         map[string]int
}

func getTestNamesToJobNamesToSkippedForAllJobs(resultsChan chan map[string]map[string]bool, jobs []*gojenkins.Job, startOfReport time.Time, ctx context.Context, jLog *log.Entry) {

	var wg sync.WaitGroup
	wg.Add(len(jobs))

	defer close(resultsChan)
	for _, job := range jobs {
		fLog := jLog.WithField("job", job.GetName())
		go getTestNamesToJobNamesToSkippedForJob(startOfReport, ctx, fLog, job, resultsChan, &wg)
	}

	wg.Wait()
	jLog.Printf("done get all jobs")
}

func getTestNamesToJobNamesToSkippedForJob(startOfReport time.Time, ctx context.Context, jLog *log.Entry, job *gojenkins.Job, resultsChan chan map[string]map[string]bool, wg *sync.WaitGroup) {
	defer wg.Done()
	testResultsForJob := flakejenkins.GetBuildNumbersToTestResultsForJob(startOfReport, job, ctx, jLog)
	testNamesToJobNamesToSkippedForJobName := map[string]map[string]bool{}
	for _, testResultForJob := range testResultsForJob {
		for _, suite := range testResultForJob.Suites {
			for _, suiteCase := range suite.Cases {
				if _, exists := testNamesToJobNamesToSkippedForJobName[suiteCase.Name]; !exists {
					testNamesToJobNamesToSkippedForJobName[suiteCase.Name] = map[string]bool{}
				}
				testNamesToJobNamesToSkippedForJobName[suiteCase.Name][job.GetName()] = suiteCase.Skipped
			}
		}
	}
	resultsChan <- testNamesToJobNamesToSkippedForJobName
}

func filterMatchingJobs(ctx context.Context, jenkins *gojenkins.Jenkins, jLog *log.Entry, opts options, innerJobs []gojenkins.InnerJob) []*gojenkins.Job {
	filteredJobs := []*gojenkins.Job{}
	jobNameRegex := regexp.MustCompile(opts.jobNamePattern)
	jLog.Printf("Filtering for jobs matching %s", jobNameRegex)
	for _, innerJob := range innerJobs {
		if !jobNameRegex.MatchString(innerJob.Name) {
			continue
		}
		job, err := jenkins.GetJob(ctx, innerJob.Name)
		if err != nil {
			jLog.Fatalf("failed to get job %s: %v", innerJob.Name, err)
		}
		filteredJobs = append(filteredJobs, job)
	}
	jLog.Printf("%d jobs left after filtering", len(filteredJobs))
	return filteredJobs
}

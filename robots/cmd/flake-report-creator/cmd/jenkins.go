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

package cmd

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	retry "github.com/avast/retry-go"
	"github.com/bndr/gojenkins"
	"github.com/joshdk/go-junit"
	log "github.com/sirupsen/logrus"
	"kubevirt.io/project-infra/robots/pkg/flakefinder"
	junitMerge "kubevirt.io/project-infra/robots/pkg/flakefinder/junit-merge"
)

const (
	defaultJenkinsBaseUrl        = "https://main-jenkins-csb-cnvqe.apps.ocp-c1.prod.psi.redhat.com/"
	defaultJenkinsJobNamePattern = "^test-kubevirt-cnv-4.10-(compute|network|operator|storage)(-[a-z0-9]+)?$"
	defaultArtifactFileNameRegex = "^(xunit_results|((merged|partial)\\.)?junit\\.functest(\\.[0-9]+)?)\\.xml$"
	JenkinsReportTemplate        = `
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
			.center {
				text-align:center
			}

			.popup {
				position: relative;
				display: inline-block;
				-webkit-user-select: none;
				-moz-user-select: none;
				-ms-user-select: none;
				user-select: none;
			}

			.popup .popuptext {
				display: none;
				width: 220px;
				background-color: #555;
				text-align: center;
				border-radius: 6px;
				padding: 8px 8px;
				position: absolute;
				z-index: 1;
				left: 50%;
				margin-left: -110px;
			}

			.popup:hover .popuptext {
				display: block;
				-webkit-animation: fadeIn 1s;
				animation: fadeIn 1s;
			}

			.nowrap {
				white-space: nowrap;
			}

			@-webkit-keyframes fadeIn {
				from {opacity: 0;}
				to {opacity: 1;}
			}

			@keyframes fadeIn {
				from {opacity: 0;}
				to {opacity:1 ;}
			}
		</style>
	</meta>
</head>
<body>
<h1>flakefinder report</h1>

<div>
	Data range from {{ $.StartOfReport }} till {{ $.EndOfReport }}<br/>
</div>

{{ if not .Headers }}
	<div>No failing tests! 🙂</div>
{{ else }}
<table>
    <tr>
        <td></td>
        <td></td>
        {{ range $header := $.Headers }}
        <td><a href="{{ $.JenkinsBaseURL }}/job/{{ $header }}/">{{ $header }}</a></td>
        {{ end }}
    </tr>
    {{ range $row, $test := $.Tests }}
    <tr>
        <td><div id="row{{$row}}"><a href="#row{{$row}}">{{ $row }}</a></div></td>
        <td>{{ $test }}</td>
        {{ range $col, $header := $.Headers }}
	        {{if not (index $.Data $test $header) }}
        <td class="center">
            N/A
        </td>
			{{else}}
        <td class="{{ (index $.Data $test $header).Severity }} center">
            <div id="r{{$row}}c{{$col}}" class="popup" >
                <span class="tests_failed" title="failed tests">{{ (index $.Data $test $header).Failed }}</span>/<span class="tests_passed" title="passed tests">{{ (index $.Data $test $header).Succeeded }}</span>/<span class="tests_skipped" title="skipped tests">{{ (index $.Data $test $header).Skipped }}</span>{{ if (index $.Data $test $header).Jobs }}
                <div class="popuptext" id="targetr{{$row}}c{{$col}}">
                    {{ range $Job := (index $.Data $test $header).Jobs }}
                    <div class="{{.Severity}} nowrap">{{ if ne .PR 0 }}<a href="{{ $.JenkinsBaseURL }}/job/{{ $header }}/{{.BuildNumber}}">{{.BuildNumber}}</a>{{ else }}<a href="{{ $.JenkinsBaseURL }}/job/{{ $header }}/{{.BuildNumber}}">{{.BuildNumber}}</a>{{ end }}</div>
                    {{ end }}
                </div>{{ end }}
            </div>
        </td>
            {{end}}
        {{ end }}
    </tr>
    {{ end }}
</table>
{{ end }}

</body>
</html>
`
	jenkinsShortUsage = "flake-report-creator jenkins creates reports from junit xml artifact files generated during jenkins builds"
)

var (
	fileNameRegex  = regexp.MustCompile(defaultArtifactFileNameRegex)
	jobNameRegexes []*regexp.Regexp
	opts           jenkinsOptions
	jenkinsCommand = &cobra.Command{
		Use:   "jenkins",
		Short: jenkinsShortUsage,
		Long: jenkinsShortUsage + `

Examples:

# create a report over all kubevirt testing jenkins jobs matching the default job name pattern for the last 14 days
$ flake-report-creator jenkins

# create a report as above for the last 3 days
$ flake-report-creator jenkins --startFrom=72h

# create a report over kubevirt testing jobs for compute and storage ocs for the given job name pattern for the last 24 hours
$ flake-report-creator jenkins --startFrom=24h --jobNamePattern='^test-kubevirt-cnv-4\.10-(compute|storage)-ocs$'
`,
		RunE: runJenkinsReport,
	}
	jLog = log.StandardLogger().WithField("type", "jenkins")
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	opts = jenkinsOptions{}
	jenkinsCommand.PersistentFlags().StringVar(&opts.endpoint, "endpoint", defaultJenkinsBaseUrl, "jenkins base url")
	jenkinsCommand.PersistentFlags().StringArrayVar(&opts.jobNamePatterns, "jobNamePattern", []string{defaultJenkinsJobNamePattern}, "jenkins job name go regex pattern to filter jobs for the report. May appear multiple times.")
	jenkinsCommand.PersistentFlags().StringVar(&opts.artifactFileNameRegex, "artifactFileNamePattern", defaultArtifactFileNameRegex, "artifact file name go regex pattern to fetch junit artifact files")
	jenkinsCommand.PersistentFlags().DurationVar(&opts.startFrom, "startFrom", 14*24*time.Hour, "The duration when the report data should be fetched")
	jenkinsCommand.PersistentFlags().IntVar(&opts.maxConnsPerHost, "maxConnsPerHost", 5, "The maximum number of connections to the endpoint (to avoid getting rate limited)")
	jenkinsCommand.PersistentFlags().BoolVar(&opts.insecureSkipVerify, "insecureSkipVerify", false, "Whether the tls verification should be skipped (this is insecure!)")
}

type jenkinsOptions struct {
	endpoint              string
	jobNamePatterns       []string
	startFrom             time.Duration
	cnvVersions           string
	maxConnsPerHost       int
	artifactFileNameRegex string
	insecureSkipVerify    bool
}

type JenkinsReportParams struct {
	flakefinder.Params
	JenkinsBaseURL string
}

type JSONParams struct {
	Data map[string]map[string]*flakefinder.Details `json:"data"`
}

func JenkinsCommand() *cobra.Command {
	return jenkinsCommand
}

func runJenkinsReport(cmd *cobra.Command, args []string) error {
	err := cmd.InheritedFlags().Parse(args)
	if err != nil {
		return err
	}

	if err = globalOpts.Validate(); err != nil {
		_, err2 := fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString(), err)
		if err2 != nil {
			return err2
		}
		return fmt.Errorf("invalid arguments provided: %v", err)
	}

	fileNameRegex, err = regexp.Compile(opts.artifactFileNameRegex)
	if err != nil {
		_, err2 := fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString(), err)
		if err2 != nil {
			return err2
		}
		return fmt.Errorf("invalid arguments provided: %v", err)
	}

	for _, jobNamePattern := range opts.jobNamePatterns {
		jobNameRegex, err := regexp.Compile(jobNamePattern)
		if err != nil {
			_, err2 := fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString(), err)
			if err2 != nil {
				return err2
			}
			jLog.Fatalf("failed to fetch jobs: %v", err)
		}
		jobNameRegexes = append(jobNameRegexes, jobNameRegex)
	}

	ctx := context.Background()

	// limit http client connections to avoid 504 errors, looks like we are getting rate limited
	client := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost: opts.maxConnsPerHost,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: opts.insecureSkipVerify,
			},
		},
	}

	jLog.Printf("Creating client for %s", opts.endpoint)
	jenkins := gojenkins.CreateJenkins(client, opts.endpoint)
	_, err = jenkins.Init(ctx)
	if err != nil {
		return fmt.Errorf("failed to contact jenkins %s: %v", opts.endpoint, err)
	}

	jLog.Printf("Fetching jobs")
	jobNames, err := jenkins.GetAllJobNames(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch jobs: %v", err)
	}
	jLog.Printf("Fetched %d jobs", len(jobNames))

	startOfReport := time.Now().Add(-1 * opts.startFrom)
	endOfReport := time.Now()

	junitReportsFromMatchingJobs := fetchJunitReportsFromMatchingJobs(startOfReport, endOfReport, jobNames, jenkins, ctx)
	writeReportToFile(startOfReport, endOfReport, junitReportsFromMatchingJobs, globalOpts.outputFile)

	return nil
}

func fetchJunitReportsFromMatchingJobs(startOfReport time.Time, endOfReport time.Time, innerJobs []gojenkins.InnerJob, jenkins *gojenkins.Jenkins, ctx context.Context) []*flakefinder.JobResult {

	filteredJobs := []gojenkins.InnerJob{}
	for _, jobNameRegex := range jobNameRegexes {
		jLog.Printf("Filtering for jobs matching %s", jobNameRegex)
		for _, innerJob := range innerJobs {
			if !jobNameRegex.MatchString(innerJob.Name) {
				continue
			}
			filteredJobs = append(filteredJobs, innerJob)
		}
	}
	jLog.Printf("%d jobs left after filtering", len(filteredJobs))

	reportChan := make(chan []*flakefinder.JobResult)

	go runReportDataFetches(filteredJobs, jenkins, ctx, startOfReport, endOfReport, reportChan)

	reports := []*flakefinder.JobResult{}
	for reportsPerJob := range reportChan {
		reports = append(reports, reportsPerJob...)
	}

	return reports
}

func runReportDataFetches(filteredJobs []gojenkins.InnerJob, jenkins *gojenkins.Jenkins, ctx context.Context, startOfReport time.Time, endOfReport time.Time, reportChan chan []*flakefinder.JobResult) {
	var wg sync.WaitGroup
	wg.Add(len(filteredJobs))

	defer close(reportChan)
	for _, filteredJob := range filteredJobs {
		go fetchReportDataForJob(filteredJob, jenkins, ctx, startOfReport, endOfReport, &wg, reportChan)
	}

	wg.Wait()
}

func fetchReportDataForJob(filteredJob gojenkins.InnerJob, jenkins *gojenkins.Jenkins, ctx context.Context, startOfReport time.Time, endOfReport time.Time, wg *sync.WaitGroup, reportChan chan []*flakefinder.JobResult) {
	defer wg.Done()

	fLog := jLog.WithField("job", filteredJob.Name)

	fLog.Printf("Fetching job")
	job, err := jenkins.GetJob(ctx, filteredJob.Name)
	if err != nil {
		fLog.Fatalf("failed to fetch job: %v", err)
	}

	completedBuilds := fetchCompletedBuildsForJob(startOfReport, job.Raw.LastBuild.Number, job, ctx, fLog)
	junitFilesFromArtifacts := fetchJunitFilesFromArtifacts(completedBuilds, fLog)
	reportsPerJob := convertJunitFileDataToReport(junitFilesFromArtifacts, ctx, job, fLog)

	reportChan <- reportsPerJob
}

func fetchCompletedBuildsForJob(startOfReport time.Time, lastBuildNumber int64, job *gojenkins.Job, ctx context.Context, fLog *log.Entry) []*gojenkins.Build {
	fLog.Printf("Fetching completed builds, starting at %d", lastBuildNumber)
	var completedBuilds []*gojenkins.Build
	for buildNumber := lastBuildNumber; buildNumber > 0; buildNumber-- {
		fLog.Printf("Fetching build no %d", buildNumber)
		build, statusCode, err := getBuildWithRetry(job, ctx, buildNumber, fLog)

		if build == nil {
			if statusCode != http.StatusNotFound {
				fLog.Fatalf("failed to fetch build data for build no %d: %v", buildNumber, err)
			}
			continue
		}

		if build.GetResult() != "SUCCESS" &&
			build.GetResult() != "UNSTABLE" {
			fLog.Printf("Skipping build with state %s", build.GetResult())
			continue
		}

		buildTime := msecsToTime(build.Info().Timestamp)
		fLog.Printf("Build %d ran at %s", build.Info().Number, buildTime.Format(time.RFC3339))
		if buildTime.Before(startOfReport) {
			fLog.Printf("Skipping remaining builds")
			break
		}

		completedBuilds = append(completedBuilds, build)
	}
	fLog.Printf("Fetched %d completed builds", len(completedBuilds))
	return completedBuilds
}

func getBuildWithRetry(job *gojenkins.Job, ctx context.Context, buildNumber int64, fLog *log.Entry) (build *gojenkins.Build, statusCode int, err error) {
	retry.Do(
		func() error {
			build, err = job.GetBuild(ctx, buildNumber)
			if err != nil {
				return err
			}
			return nil
		},
		retry.RetryIf(func(err error) bool {
			fLog.Warningf("failed to fetch build data for build no %d: %v", buildNumber, err)
			statusCode = httpStatusOrDie(err, fLog)
			if statusCode == http.StatusNotFound {
				return false
			}
			if statusCode == http.StatusGatewayTimeout {
				return true
			}
			return false
		}),
	)
	return build, statusCode, err
}

// httpStatusOrDie fetches [stringly typed](https://wiki.c2.com/?StringlyTyped) error code produced by jenkins client
// or logs a fatal error if conversion to int is not possible
func httpStatusOrDie(err error, fLog *log.Entry) int {
	statusCode, conversionError := strconv.Atoi(err.Error())
	if conversionError != nil {
		fLog.Fatalf("Failed to get status code from error %v: %v", err, conversionError)
	}
	return statusCode
}

func msecsToTime(msecs int64) time.Time {
	return time.Unix(msecs/1000, msecs%1000)
}

func fetchJunitFilesFromArtifacts(completedBuilds []*gojenkins.Build, fLog *log.Entry) []gojenkins.Artifact {
	fLog.Printf("Fetch junit files from artifacts for %d completed builds", len(completedBuilds))
	artifacts := []gojenkins.Artifact{}
	for _, completedBuild := range completedBuilds {
		for _, artifact := range completedBuild.GetArtifacts() {
			if !fileNameRegex.MatchString(artifact.FileName) {
				continue
			}
			artifacts = append(artifacts, artifact)
		}
	}
	fLog.Printf("Fetched %d junit files from artifacts", len(artifacts))
	return artifacts
}

func convertJunitFileDataToReport(junitFilesFromArtifacts []gojenkins.Artifact, ctx context.Context, job *gojenkins.Job, fLog *log.Entry) []*flakefinder.JobResult {

	// problem: we might encounter multiple junit artifacts per job run, we need to merge them into
	// 			one so the report builder can handle the results

	// step 1: download the report junit data and store them in a slice per build
	fLog.Printf("Download %d artifacts and convert to reports", len(junitFilesFromArtifacts))
	artifactsPerBuild := map[int64][][]junit.Suite{}
	for _, artifact := range junitFilesFromArtifacts {
		data, err := artifact.GetData(ctx)
		if err != nil {
			fLog.Fatalf("failed to fetch artifact data: %v", err)
		}
		report, err := junit.Ingest(data)
		if err != nil {
			fLog.Fatalf("failed to fetch artifact data: %v", err)
		}
		buildNumber := artifact.Build.Info().Number
		artifactsPerBuild[buildNumber] = append(artifactsPerBuild[buildNumber], report)
	}

	// step 2: merge all the suites for a build into one suite per build
	fLog.Printf("Merge reports for %d builds", len(artifactsPerBuild))
	reportsPerJob := []*flakefinder.JobResult{}
	for buildNumber, artifacts := range artifactsPerBuild {
		// TODO: evaluate conflicts somehow
		mergedResult, _ := junitMerge.Merge(artifacts)
		reportsPerJob = append(reportsPerJob, &flakefinder.JobResult{Job: job.GetName(), JUnit: mergedResult, BuildNumber: int(buildNumber)})
	}

	return reportsPerJob
}

func writeReportToFile(startOfReport time.Time, endOfReport time.Time, reports []*flakefinder.JobResult, outputFile string) {
	parameters := flakefinder.CreateFlakeReportData(reports, []int{}, endOfReport, "kubevirt", "kubevirt", startOfReport)
	jLog.Printf("writing output to %s", outputFile)

	writeHTMLReportToOutputFile(outputFile, JenkinsReportTemplate, JenkinsReportParams{Params: parameters, JenkinsBaseURL: opts.endpoint})

	writeJSONToOutputFile(strings.TrimSuffix(outputFile, ".html")+".json", parameters)
}

func writeHTMLReportToOutputFile(outputFile string, reportTemplate string, params interface{}) {
	reportOutputWriter := createReportOutputWriter(outputFile)
	defer reportOutputWriter.Close()

	err := flakefinder.WriteTemplateToOutput(reportTemplate, params, reportOutputWriter)
	if err != nil {
		jLog.Fatalf("failed to write report: %v", err)
	}
}

func writeJSONToOutputFile(jsonOutputFile string, parameters flakefinder.Params) {
	reportOutputWriter := createReportOutputWriter(jsonOutputFile)
	defer reportOutputWriter.Close()

	encoder := json.NewEncoder(reportOutputWriter)
	err := encoder.Encode(parameters.Data)
	if err != nil {
		jLog.Fatalf("failed to write report: %v", err)
	}
}

func createReportOutputWriter(outputFile string) *os.File {
	reportOutputWriter, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil && err != os.ErrNotExist {
		jLog.Fatalf("failed to write report: %v", err)
	}
	return reportOutputWriter
}

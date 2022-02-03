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
	"flag"
	"fmt"
	"github.com/bndr/gojenkins"
	"github.com/joshdk/go-junit"
	junit_merge "kubevirt.io/project-infra/robots/cmd/jenkins-artifact-retriever/junit-merge"
	"kubevirt.io/project-infra/robots/pkg/flakefinder"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"
)

const (
	defaultJenkinsBaseUrl        = "https://main-jenkins-csb-cnvqe.apps.ocp-c1.prod.psi.redhat.com/"
	defaultJenkinsJobNamePattern = "^test-kubevirt-cnv-%s-(compute|network|operator|storage)(-[a-z0-9]+)?$"
	defaultCNVVersions = "4.10"
	ReportTemplate               = `
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
        .right {
            text-align: right;
			width: 100%;
        }


        /* Popup container - can be anything you want */
        .popup {
            position: relative;
            display: inline-block;
            cursor: pointer;
            -webkit-user-select: none;
            -moz-user-select: none;
            -ms-user-select: none;
            user-select: none;
        }

        /* The actual popup */
        .popup .popuptext {
            visibility: hidden;
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

        .nowrap {
            white-space: nowrap;
        }

        /* Toggle this class - hide and show the popup */
        .popup .show {
            visibility: visible;
            -webkit-animation: fadeIn 1s;
            animation: fadeIn 1s;
        }

        /* Add animation (fade in the popup) */
        @-webkit-keyframes fadeIn {
            from {opacity: 0;}
            to {opacity: 1;}
        }

        @keyframes fadeIn {
            from {opacity: 0;}
            to {opacity:1 ;}
        }

	</style>
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
        <td><div id="row{{$row}}"><a href="#row{{$row}}">{{ $row }}</a><div></td>
        <td>{{ $test }}</td>
        {{ range $col, $header := $.Headers }}
        {{if not (index $.Data $test $header) }}
        <td class="center">
            N/A
        </td>
        {{else}}
        <td class="{{ (index $.Data $test $header).Severity }} center">
            <div id="r{{$row}}c{{$col}}" onClick="popup(this.id)" class="popup" >
                <span class="tests_failed" title="failed tests">{{ (index $.Data $test $header).Failed }}</span>/<span class="tests_passed" title="passed tests">{{ (index $.Data $test $header).Succeeded }}</span>/<span class="tests_skipped" title="skipped tests">{{ (index $.Data $test $header).Skipped }}</span>
                <div class="popuptext" id="targetr{{$row}}c{{$col}}">
                    {{ range $Job := (index $.Data $test $header).Jobs }}
                    <div class="{{.Severity}} nowrap">{{ if ne .PR 0 }}<a href="{{ $.JenkinsBaseURL }}/job/{{ $header }}/{{.BuildNumber}}">{{.BuildNumber}}</a>{{ else }}<a href="{{ $.JenkinsBaseURL }}/job/{{ $header }}/{{.BuildNumber}}">{{.BuildNumber}}</a>{{ end }}</div>
                    {{ end }}
                </div>
            </div>
            {{end}}
        </td>
        {{ end }}
    </tr>
    {{ end }}
</table>
{{ end }}

<script>
    function popup(id) {
        var popup = document.getElementById("target" + id);
        popup.classList.toggle("show");
    }
</script>

</body>
</html>
`
)

var (
	fileNameRegex *regexp.Regexp
	opts          options
)

func init() {
	fileNameRegex = regexp.MustCompile("^(partial\\.)junit\\.functest(\\.1)\\.xml$")
}

func flagOptions() options {
	o := options{}
	flag.StringVar(&o.endpoint, "endpoint", defaultJenkinsBaseUrl, "jenkins base url")
	flag.StringVar(&o.jobNamePattern, "jobNamePattern", defaultJenkinsJobNamePattern, "jenkins job name pattern to filter jobs for for the report (note the pattern that is used for injecting the CNV version)")
	flag.StringVar(&o.cnvVersions, "cnvVersions", defaultCNVVersions, "comma separated list of cnv versions to report")
	flag.DurationVar(&o.startFrom, "startFrom", 14*24*time.Hour, "The duration when the report data should be fetched")
	flag.Parse()
	return o
}

type options struct {
	endpoint       string
	jobNamePattern string
	startFrom      time.Duration
	cnvVersions    string
}

type JenkinsReportParams struct {
	flakefinder.Params
	JenkinsBaseURL string
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("jenkins-flake-reporter")
	opts = flagOptions()
	RunJenkinsReport(defaultJenkinsBaseUrl, opts.cnvVersions)
}

func RunJenkinsReport(jenkinsBaseUrl string, cnvVersions ...string) {
	ctx := context.Background()

	// limit http client connections to avoid 504 errors, looks like we are getting rate limited
	client := &http.Client{
		Transport:     &http.Transport{
			MaxConnsPerHost: 5,
		},
	}

	log.Printf("Creating client for %s", jenkinsBaseUrl)
	jenkins := gojenkins.CreateJenkins(client, jenkinsBaseUrl)
	_, err := jenkins.Init(ctx)
	if err != nil {
		log.Fatalf("failed to contact jenkins %s: %v", jenkinsBaseUrl, err)
	}

	log.Printf("Fetching jobs")
	jobNames, err := jenkins.GetAllJobNames(ctx)
	if err != nil {
		log.Fatalf("failed to fetch jobs: %v", err)
	}
	log.Printf("Fetched %d jobs", len(jobNames))

	startOfReport := time.Now().Add(-1 * opts.startFrom)
	endOfReport := time.Now()

	junitReportsFromMatchingJobs := fetchJunitReportsFromMatchingJobs(startOfReport, endOfReport, cnvVersions, jobNames, jenkins, ctx)
	writeReportToFile(startOfReport, endOfReport, junitReportsFromMatchingJobs)

}

func fetchJunitReportsFromMatchingJobs(startOfReport time.Time, endOfReport time.Time, cnvVersions []string, jobNames []gojenkins.InnerJob, jenkins *gojenkins.Jenkins, ctx context.Context) []*flakefinder.JobResult {
	reports := []*flakefinder.JobResult{}

	reportChan := make(chan []*flakefinder.JobResult)
	var wg sync.WaitGroup

	for _, cnvVersion := range cnvVersions {
		log.Printf("Filtering jobs for CNV %s matching %s", cnvVersion, opts.jobNamePattern)
		compile, err := regexp.Compile(fmt.Sprintf(opts.jobNamePattern, cnvVersion))
		if err != nil {
			log.Fatalf("failed to fetch jobs: %v", err)
		}
		for _, jobName := range jobNames {
			if !compile.MatchString(jobName.Name) {
				continue
			}

			wg.Add(1)
			go func(jobName gojenkins.InnerJob, ctx context.Context) {
				log.Printf("Job %s matches", jobName)

				log.Printf("Fetching job %s", jobName)
				job, err := jenkins.GetJob(ctx, jobName.Name)
				if err != nil {
					log.Fatalf("failed to fetch job %s: %v", jobName, err)
				}

				lastBuild, err := job.GetLastBuild(ctx)
				if err != nil {
					log.Fatalf("failed to fetch last build for job %s: %v", jobName, err)
				}

				completedBuilds := fetchCompletedBuildsForJob(startOfReport, endOfReport, lastBuild, job, ctx)
				junitFilesFromArtifacts := fetchJunitFilesFromArtifacts(completedBuilds)
				reportsPerJob := convertJunitFileDataToReport(junitFilesFromArtifacts, ctx, job)

				reportChan <- reportsPerJob
			}(jobName, ctx)
		}
	}

	go func() {
		for reportsPerJob := range reportChan {
			reports = append(reports, reportsPerJob...)
			wg.Done()
		}
	}()

	wg.Wait()
	return reports
}

func fetchCompletedBuildsForJob(startOfReport time.Time, endOfReport time.Time, lastBuild *gojenkins.Build, job *gojenkins.Job, ctx context.Context) []*gojenkins.Build {
	log.Printf("Fetching completed builds from %s - %s period", startOfReport.Format(time.RFC3339), endOfReport.Format(time.RFC3339))
	var completedBuilds []*gojenkins.Build
	for i := lastBuild.GetBuildNumber(); i > 0; i-- {
		log.Printf("Fetching build no %d", i)
		build, err := job.GetBuild(ctx, i)
		if err != nil {
			log.Fatalf("failed to fetch build data: %v", err)
		}

		if build.GetResult() != "SUCCESS" &&
			build.GetResult() != "UNSTABLE" {
			log.Printf("Skipping %s builds", build.GetResult())
			continue
		}

		buildTime := msecsToTime(build.Info().Timestamp)
		log.Printf("Build %d ran at %s", build.Info().Number, buildTime.Format(time.RFC3339))
		if buildTime.Before(startOfReport) {
			log.Printf("Skipping remaining builds for %s", job.GetName())
			break
		}

		completedBuilds = append(completedBuilds, build)
	}
	log.Printf("Fetched %d completed builds from %s - %s period", len(completedBuilds), startOfReport, endOfReport)
	return completedBuilds
}

func msecsToTime(msecs int64) time.Time {
	return time.Unix(msecs / 1000, msecs % 1000)
}

func fetchJunitFilesFromArtifacts(completedBuilds []*gojenkins.Build) []gojenkins.Artifact {
	log.Printf("Fetch junit files from artifacts for %d completed builds", len(completedBuilds))
	artifacts := []gojenkins.Artifact{}
	for _, completedBuild := range completedBuilds {
		for _, artifact := range completedBuild.GetArtifacts() {
			if !fileNameRegex.MatchString(artifact.FileName) {
				continue
			}
			artifacts = append(artifacts, artifact)
		}
	}
	log.Printf("Fetched %d junit files from artifacts", len(artifacts))
	return artifacts
}

func convertJunitFileDataToReport(junitFilesFromArtifacts []gojenkins.Artifact, ctx context.Context, job *gojenkins.Job) []*flakefinder.JobResult {

	// problem: we might encounter multiple junit artifacts per job run, we need to merge them into
	// 			one so the report builder can handle the results

	// step 1: download the report junit data and store them in a slice per build
	artifactsPerBuild := map[int64][][]junit.Suite{}
	for _, artifact := range junitFilesFromArtifacts {
		data, err := artifact.GetData(ctx)
		if err != nil {
			log.Fatalf("failed to fetch artifact data: %v", err)
		}
		report, err := junit.Ingest(data)
		if err != nil {
			log.Fatalf("failed to fetch artifact data: %v", err)
		}
		buildNumber := artifact.Build.Info().Number
		artifactsPerBuild[buildNumber] = append(artifactsPerBuild[buildNumber], report)
	}

	// step 2: merge all the suites for a build into one suite per build
	reportsPerJob := []*flakefinder.JobResult{}
	for buildNumber, artifacts := range artifactsPerBuild {
		// TODO: evaluate conflicts somehow
		mergedResult, _ := junit_merge.Merge(artifacts)
		reportsPerJob = append(reportsPerJob, &flakefinder.JobResult{Job: job.GetName(), JUnit: mergedResult, BuildNumber: int(buildNumber)})
	}

	return reportsPerJob
}

func writeReportToFile(startOfReport time.Time, endOfReport time.Time, reports []*flakefinder.JobResult) {
	parameters := flakefinder.CreateFlakeReportData(reports, []int{}, endOfReport, "kubevirt", "kubevirt", startOfReport)

	outputFile, err := os.CreateTemp("", "flakefinder-*.html")
	if err != nil {
		log.Fatalf("failed to write report: %v", err)
	}
	log.Printf("writing output to %s", outputFile.Name())

	reportOutputWriter, err := os.OpenFile(outputFile.Name(), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil && err != os.ErrNotExist {
		log.Fatalf("failed to write report: %v", err)
	}
	defer reportOutputWriter.Close()

	err = flakefinder.WriteTemplateToOutput(ReportTemplate, JenkinsReportParams{Params: parameters, JenkinsBaseURL: opts.endpoint}, reportOutputWriter)
	if err != nil {
		log.Fatalf("failed to write report: %v", err)
	}
}

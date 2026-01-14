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

package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/avast/retry-go"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
	"kubevirt.io/project-infra/pkg/flakefinder"
)

const defaultDaysInThePast = 7
const defaultOrg = "kubevirt"
const defaultRepo = "kubevirt"

type TemplateData struct {
	DaysInThePast                 int
	Date                          time.Time
	PrNumbersToJobs               map[string][]Job
	Org                           string
	Repo                          string
	PrNumbersSortedByFailuresDesc []string
}

func (t *TemplateData) AttachSortedInformation() {
	for key := range t.PrNumbersToJobs {
		t.PrNumbersSortedByFailuresDesc = append(t.PrNumbersSortedByFailuresDesc, key)
	}
	sort.Slice(t.PrNumbersSortedByFailuresDesc, func(i, j int) bool {
		return len(t.PrNumbersToJobs[t.PrNumbersSortedByFailuresDesc[i]]) > len(t.PrNumbersToJobs[t.PrNumbersSortedByFailuresDesc[j]])
	})
}

type Job struct {
	JobName      string
	BuildNumber  string
	BuildURL     string
	Failure      bool
	ArtifactsURL string
}

type options struct {
	daysInThePast       int
	outputFile          string
	overwriteOutputFile bool
	org                 string
	repo                string
}

func (o *options) validate() error {
	if o.daysInThePast <= 0 {
		return fmt.Errorf("invalid value for daysInThePast %d", o.daysInThePast)
	}
	if o.outputFile == "" {
		file, err := os.CreateTemp("", "merge-commit-summary-*.html")
		if err != nil {
			return fmt.Errorf("failed to generate temp file: %w", err)
		}
		o.outputFile = file.Name()
	} else {
		if !o.overwriteOutputFile {
			stats, err := os.Stat(o.outputFile)
			if stats != nil || !os.IsNotExist(err) {
				return fmt.Errorf("file %q exists or error occurred: %w", o.outputFile, err)
			}
		}
	}
	return nil
}

func (o *options) parseFlags() {
	flag.IntVar(&o.daysInThePast, "days-in-the-past", defaultDaysInThePast, "determines how much days in the past till today are covered")
	flag.StringVar(&o.outputFile, "output-file", "", "outputfile to write to, default is a tempfile in folder")
	flag.BoolVar(&o.overwriteOutputFile, "overwrite-output-file", false, "whether outputfile is set to be overwritten if it exists")
	flag.StringVar(&o.org, "org", defaultOrg, "GitHub org to use for fetching report data from gcs dir")
	flag.StringVar(&o.repo, "repo", defaultRepo, "GitHub repo to use for fetching report data from gcs dir")
	flag.Parse()
}

var (
	//go:embed merge-commit-summary.gohtml
	htmlTemplate string

	opts = options{}
)

func init() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.JSONFormatter{})
}

func main() {
	opts.parseFlags()
	err := opts.validate()
	if err != nil {
		log.Fatalf("failed to validate flags: %v", err)
	}

	// 1.) fetch merged PRs from last x days
	// we're not using the github api client here as it's much easier searching for a bunch of PRs with cli
	dateFrom := time.Now().Add(time.Duration(-1*opts.daysInThePast*24) * time.Hour).Format(time.DateOnly)
	ghCommand := exec.Command("gh", "search", "prs", "--repo=kubevirt/kubevirt", "--merged", "--json", "number", "--jq", ".[].number", "--", fmt.Sprintf(`/retest updated:>=%s`, dateFrom))
	output, err := ghCommand.Output()
	if err != nil {
		log.Fatalf("failed to execute gh command %q\n%s\n%v", ghCommand, output, err)
	}
	log.Debugf("command %q executed\noutput: %s", ghCommand, output)
	prNumbers := string(output)
	prNumberStrings := strings.Split(prNumbers, "\n")

	// 2.) get failed builds for latest commit per PR
	log.Debugf("fetching data for PRs %v", prNumberStrings)
	prsToJobs := make(map[string][]Job)
	for _, prNumberString := range prNumberStrings {
		if prNumberString == "" {
			continue
		}
		jobsLatestCommit, err := getJobsForLatestCommit(opts.org, opts.repo, prNumberString)
		if err != nil {
			log.Fatalf("failed to get job for PR %s: %v", prNumberString, err)
		}
		prsToJobs[prNumberString] = jobsLatestCommit
	}

	htmlReportOutputWriter, err := os.Create(opts.outputFile)
	if err != nil {
		log.WithError(err).Fatalf("failed to create file %q", opts.outputFile)
	}
	log.Printf("Writing html to %q", opts.outputFile)
	defer func() {
		err2 := htmlReportOutputWriter.Close()
		if err2 != nil {
			log.WithError(err2).Error("failed to close html report output writer")
		}
	}()

	templateData := &TemplateData{
		DaysInThePast:   opts.daysInThePast,
		Date:            time.Now(),
		PrNumbersToJobs: prsToJobs,
		Org:             opts.org,
		Repo:            opts.repo,
	}
	templateData.AttachSortedInformation()
	log.Debugf("templateData: %+v", templateData)
	err = flakefinder.WriteTemplateToOutput(htmlTemplate, templateData, htmlReportOutputWriter)
	if err != nil {
		log.Fatalf("failed to write to file %q: %+v", opts.outputFile, err)
	}
}

func getJobsForLatestCommit(org string, repo string, prNumber string) (jobsLatestCommit []Job, err error) {
	prHistory := prHistoryURL(org, repo, prNumber)
	resp, err := httpGetWithRetry(prHistory)
	if err != nil {
		return nil, err
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			log.WithError(err).Error("failed to close body")
		}
	}()

	prHistoryPage, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing history page: %s", err)
	}
	latestCommit := getLatestCommit(prHistoryPage)
	if latestCommit == "" {
		return nil, fmt.Errorf("failed to get latest commit from %s", prHistory)
	}
	jobsAllCommits := filterJobs(org, repo, prHistoryPage)

	jobsLatestCommit, err = filterForLastCommit(org, repo, prNumber, latestCommit, jobsAllCommits)
	if err != nil {
		return nil, err
	}
	return jobsLatestCommit, nil
}

func getLatestCommit(node *html.Node) (latestCommit string) {
	if node.Type == html.ElementNode && node.Data == "a" {
		for _, td := range node.Attr {
			if strings.Contains(td.Val, "commit") {
				commitURL := strings.Split(td.Val, "/")
				latestCommit := commitURL[len(commitURL)-1]
				return latestCommit
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		latestCommit = getLatestCommit(child)
		if latestCommit != "" {
			return latestCommit
		}
	}
	return ""
}

func filterJobs(org string, repo string, node *html.Node) (jobs []Job) {
	var e2eJob Job
	if node.Type == html.ElementNode && node.Data == "td" {
		for _, td := range node.Attr {
			if !strings.Contains(td.Val, "run-failure") {
				continue
			}
			for _, href := range node.FirstChild.Attr {
				if strings.Contains(href.Val, "e2e") {
					e2eJob.Failure = true
					e2eJob.BuildURL = fmt.Sprintf("https://prow.ci.kubevirt.io/%s", href.Val)
					buildLogUrl := strings.Split(href.Val, "/")
					e2eJob.JobName = buildLogUrl[len(buildLogUrl)-2]
					e2eJob.BuildNumber = buildLogUrl[len(buildLogUrl)-1]
					prNumber := buildLogUrl[len(buildLogUrl)-3]
					e2eJob.ArtifactsURL = fmt.Sprintf("https://gcsweb.ci.kubevirt.io/gcs/kubevirt-prow/pr-logs/pull/%s_%s/%s/%s/%s/artifacts", org, repo, prNumber, e2eJob.JobName, e2eJob.BuildNumber)
					jobs = append(jobs, e2eJob)
					continue
				}
			}
		}
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		childJobs := filterJobs(org, repo, child)
		jobs = append(jobs, childJobs...)
	}
	return jobs
}

func filterForLastCommit(org string, repo string, prNumber string, latestCommit string, jobList []Job) ([]Job, error) {
	var filteredJobList []Job
	for _, job := range jobList {
		finishedJSON, err := httpGetWithRetry(finishedJSONURL(org, repo, prNumber, job.JobName, job.BuildNumber))
		if err != nil {
			return nil, fmt.Errorf("failed to get %s finished.json : %s", job.JobName, err)
		}
		if finishedJSON.StatusCode != http.StatusOK {
			continue
		}
		defer func() {
			err2 := finishedJSON.Body.Close()
			if err2 != nil {
				log.WithError(err2).Error("failed to close finished json body")
			}
		}()

		finishedJSONData, err := io.ReadAll(finishedJSON.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read finished JSON for %s -- %s", job.JobName, err)
		}
		var data map[string]interface{}
		err = json.Unmarshal(finishedJSONData, &data)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshall finished JSON for %s -- %s", job.JobName, err)
		}
		if latestCommit == data["revision"] {
			filteredJobList = append(filteredJobList, job)
		}
	}
	return filteredJobList, nil
}

func httpGetWithRetry(url string) (resp *http.Response, err error) {
	httpRetryLog := log.WithField("url", url)
	err = retry.Do(
		func() error {
			resp, err = http.Get(url)
			switch {
			case resp.StatusCode == http.StatusOK:
				httpRetryLog.Debugf("http get succeeded")
				return nil
			case resp.StatusCode == http.StatusGatewayTimeout:
				httpRetryLog.Debugf("failed to http get, will retry")
				return fmt.Errorf("failed to http get %s (status %d): %v", url, resp.StatusCode, err)
			case err != nil:
				httpRetryLog.Debugf("failed to http get, aborting")
				return retry.Unrecoverable(err)
			default:
				httpRetryLog.Debugf("failed to http get, aborting")
				return retry.Unrecoverable(fmt.Errorf("failed to http get %s (status %d): %v", url, resp.StatusCode, err))
			}
		},
	)
	return
}

func prHistoryURL(org string, repo string, prNumber string) string {
	return fmt.Sprintf("https://prow.ci.kubevirt.io/pr-history/?org=%s&repo=%s&pr=%s", org, repo, prNumber)
}

func finishedJSONURL(org string, repo string, prNumber string, jobName string, buildNumber string) string {
	return fmt.Sprintf("%s/%s/%s/finished.json", fmt.Sprintf("https://gcsweb.ci.kubevirt.io/gcs/kubevirt-prow/pr-logs/pull/%s_%s/%s/", org, repo, prNumber), jobName, buildNumber)
}

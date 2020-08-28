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
 * Copyright 2019 Red Hat, Inc.
 *
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"k8s.io/test-infra/prow/config/secret"
	"k8s.io/test-infra/prow/flagutil"

	"kubevirt.io/project-infra/robots/pkg/flakefinder"
)

func flagOptions() options {
	o := options{
		endpoint: flagutil.NewStrings("https://api.github.com"),
	}
	flag.BoolVar(&o.isDryRun, "dry-run", true, "Whether report should be only printed to standard out instead of written to gcs") // TODO: incompatible change, requires setting flags on jobs
	flag.IntVar(&o.ceiling, "ceiling", 100, "Maximum number of issues to modify, 0 for infinite")
	flag.DurationVar(&o.merged, "merged", 24*7*time.Hour, "Filter to issues merged in the time window")
	flag.Var(&o.endpoint, "endpoint", "GitHub's API endpoint")
	flag.StringVar(&o.token, "token", "", "Path to github token")
	flag.BoolVar(&o.isPreview, "preview", false, "Whether report should be written to preview directory")
	flag.StringVar(&o.prBaseBranch, "pr_base_branch", PRBaseBranchDefault, "Base branch for the PRs")
	flag.StringVar(&o.reportOutputChildPath, "report_output_child_path", "", fmt.Sprintf("Child path below the main reporting directory '%s' (i.e. 'master')", flakefinder.ReportsPath))
	flag.StringVar(&o.org, "org", Org, "GitHub org name")
	flag.StringVar(&o.repo, "repo", Repo, "GitHub org name")
	flag.BoolVar(&o.today, "today", false, "Whether to create a report for the current day only (i.e. using data starting from report day 00:00Z till now)")
	flag.BoolVar(&o.skipBeforeStartOfReport, "skip_results_before_start_of_report", true, "Whether to skip test results occurring before start of report")
	flag.BoolVar(&o.stdout, "stdout", false, "(Deprecated, use dry-run instead) write generated report to stdout")
	flag.Parse()
	return o
}

type options struct {
	isDryRun bool

	// Deprecated: no function
	ceiling               int
	endpoint              flagutil.Strings
	token                 string
	graphqlEndpoint       string
	merged                time.Duration
	isPreview             bool
	prBaseBranch          string
	reportOutputChildPath string
	org                   string
	repo                  string

	// Deprecated: replaced by dry-run
	stdout bool
	today  bool

	skipBeforeStartOfReport bool
}

const MaxNumberOfReportsToLinkTo = 50
const PRBaseBranchDefault = "master"
const Org = "kubevirt"
const Repo = "kubevirt"

var ReportOutputPath = flakefinder.ReportsPath
var PRBaseBranch string

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	o := flagOptions()

	if o.token == "" {
		log.Fatal("empty --token")
	}

	ReportOutputPath = BuildReportOutputPath(o)
	PRBaseBranch = o.prBaseBranch

	secretAgent := &secret.Agent{}
	if err := secretAgent.Start([]string{o.token}); err != nil {
		log.Fatalf("Error starting secrets agent: %v", err)
	}

	var err error
	for _, ep := range o.endpoint.Strings() {
		_, err = url.ParseRequestURI(ep)
		if err != nil {
			log.Fatalf("Invalid --endpoint URL %q: %v.", ep, err)
		}
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: string(secretAgent.GetSecret(o.token))},
	)
	tc := oauth2.NewClient(ctx, ts)

	c := github.NewClient(tc)

	endOfReport := time.Now()
	startOfReport, endOfReport := GetReportInterval(o, endOfReport)
	logrus.Infof("Fetching Prs from %v till %v", startOfReport, endOfReport)

	logrus.Infof("Filtering PRs for base branch %s", PRBaseBranch)
	prs := []*github.PullRequest{}
	prNumbers := []int{}
	for nextPage := 1; nextPage > 0; {
		pullRequests, response, err := c.PullRequests.List(ctx, o.org, o.repo, &github.PullRequestListOptions{
			Base:        PRBaseBranch,
			State:       "closed",
			Sort:        "updated",
			Direction:   "desc",
			ListOptions: github.ListOptions{Page: nextPage},
		})
		if err != nil {
			log.Fatalf("Failed to fetch PRs for page %d: %v.", nextPage, err)
		}
		nextPage = response.NextPage
		for _, pr := range pullRequests {
			if startOfReport.After(*pr.UpdatedAt) {
				nextPage = 0
				break
			}
			if pr.MergedAt == nil {
				continue
			}
			if startOfReport.After(*pr.MergedAt) {
				continue
			}
			if endOfReport.Before(*pr.MergedAt) {
				continue
			}
			logrus.Infof("Adding PR %v '%v' (updated at %s)", *pr.Number, *pr.Title, pr.UpdatedAt.Format(time.RFC3339))
			prs = append(prs, pr)
			prNumbers = append(prNumbers, *pr.Number)
		}
	}
	logrus.Infof("%d pull requests found.", len(prs))

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create new storage client: %v.\n", err)
	}
	reports := []*Result{}
	for _, pr := range prs {
		r, err := FindUnitTestFiles(ctx, client, flakefinder.BucketName, strings.Join([]string{o.org, o.repo}, "/"), pr, startOfReport, o.skipBeforeStartOfReport)
		if err != nil {
			log.Printf("failed to load JUnit file for %v: %v", pr.Number, err)
		}
		reports = append(reports, r...)
	}

	err = WriteReportToBucket(ctx, client, reports, o.merged, o.org, o.repo, prNumbers, o.stdout, o.isDryRun, startOfReport, endOfReport)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to write report: %v", err))
		return
	}

	printIndexPageToStdOut := o.isDryRun && o.stdout
	err = CreateReportIndex(ctx, client, o.org, o.repo, printIndexPageToStdOut)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create report index page: %v", err))
		return
	}
}

func GetReportInterval(o options, till time.Time) (startOfReport, endOfReport time.Time) {
	if o.today {
		startOfReportToday := till.Format("2006-01-02") + "T00:00:00Z"
		startOfReport, err := time.Parse(time.RFC3339, startOfReportToday)
		if err != nil {
			log.Fatalf("Failed to parse time %+v: %+v", startOfReportToday, err)
		}
		return startOfReport, till
	} else {
		startOfReport = till.Add(-o.merged)
	}

	// we normalize the start of the report against start of day vs. start of the hour to avoid working against a
	// moving target.
	// In general a user would expect to find all pull requests of the previous day in a 24h report, regardless of
	// when the report has been run at the current day, which, depending on time of day when the report had been run,
	// would not always be the case.
	// Consider i.e. if the report is run late in the afternoon the user might wonder why the PR merged in the morning
	// the day before was not included.

	var startOfDayOrHour, endOfDayOrHour string

	// in case of reports for at least a day we are fetching reports from start of previous day till end of that day
	if o.merged.Hours() < 24 {
		// in case of less than a day we are fetching reports from start of the hour
		startOfDayOrHour = startOfReport.Format("2006-01-02T15:00:00Z07:00")
		endOfDayOrHour = till.Format("2006-01-02T15:00:00Z07:00")
	} else {
		startOfDayOrHour = startOfReport.Format("2006-01-02") + "T00:00:00Z"
		endOfDayOrHour = till.Format("2006-01-02") + "T00:00:00Z"
	}
	startOfReport, err := time.Parse(time.RFC3339, startOfDayOrHour)
	if err != nil {
		log.Fatalf("Failed to parse time %+v: %+v", startOfDayOrHour, err)
	}
	endOfReport, err = time.Parse(time.RFC3339, endOfDayOrHour)
	if err != nil {
		log.Fatalf("Failed to parse time %+v: %+v", endOfDayOrHour, err)
	}
	millisecond, err := time.ParseDuration("1ms")
	if err != nil {
		log.Fatalf("Failed to parse duration 1ms: %+v", err)
	}
	endOfReport = endOfReport.Add(-millisecond)
	return startOfReport, endOfReport
}

// BuildReportOutputPath creates the path to which the report will get written, considering also if we are in
// preview mode, so that existing production reports will not be overwritten. I.e considering
// options{
//		reportOutputChildPath: "kubevirt/kubevirt"
//		isPreview:			   true
// }
// will lead to
// "reports/flakefinder/preview/kubevirt/kubevirt"
//
func BuildReportOutputPath(o options) string {
	outputPath := flakefinder.ReportsPath
	if o.isPreview {
		outputPath = filepath.Join(outputPath, flakefinder.PreviewPath)
	}
	outputPath = filepath.Join(outputPath, o.reportOutputChildPath)
	return outputPath
}

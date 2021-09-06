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
	"k8s.io/test-infra/prow/config/secret"
	"log"
	"net/url"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"
	"k8s.io/test-infra/prow/flagutil"

	"kubevirt.io/project-infra/robots/pkg/flakefinder"
)

func flagOptions() options {
	o := options{
		endpoint: flagutil.NewStrings("https://api.github.com"),
	}
	flag.BoolVar(&o.isDryRun, "dry-run", true, "Whether report should be only printed to standard out instead of written to gcs") // TODO: incompatible change, requires setting flags on jobs
	flag.DurationVar(&o.merged, "merged", 24*7*time.Hour, "Filter to issues merged in the time window")
	flag.Var(&o.endpoint, "endpoint", "GitHub's API endpoint")
	flag.StringVar(&o.tokenPath, "token", "", "Path to github token")
	flag.BoolVar(&o.isPreview, "preview", false, "Whether report should be written to preview directory")
	flag.StringVar(&o.prBaseBranch, "pr_base_branch", PRBaseBranchDefault, "Base branch for the PRs")
	flag.StringVar(&o.reportOutputChildPath, "report_output_child_path", "", fmt.Sprintf("Child path below the main reporting directory '%s' (i.e. 'master')", flakefinder.ReportsPath))
	flag.StringVar(&o.org, "org", Org, "GitHub org name")
	flag.StringVar(&o.repo, "repo", Repo, "GitHub org name")
	flag.BoolVar(&o.today, "today", false, "Whether to create a report for the current day only (i.e. using data starting from report day 00:00Z till now)")
	flag.BoolVar(&o.skipBeforeStartOfReport, "skip_results_before_start_of_report", true, "Whether to skip test results occurring before start of report")
	flag.StringVar(&o.periodicJobDirRegex, "periodic_job_dir_regex", "", "Regular expression to use for fetching data from periodic jobs, or empty string if not wanted")
	flag.Parse()
	return o
}

type options struct {
	isDryRun                bool
	endpoint                flagutil.Strings
	tokenPath               string
	graphqlEndpoint         string
	merged                  time.Duration
	isPreview               bool
	prBaseBranch            string
	reportOutputChildPath   string
	org                     string
	repo                    string
	today                   bool
	skipBeforeStartOfReport bool
	periodicJobDirRegex     string
}

const MaxNumberOfReportsToLinkTo = 50
const PRBaseBranchDefault = "master"
const Org = "kubevirt"
const Repo = "kubevirt"

var ReportOutputPath = flakefinder.ReportsPath

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	o := flagOptions()

	if o.tokenPath == "" {
		log.Fatal("empty --token")
	}
	err := secret.Add(o.tokenPath)
	if err != nil {
		log.Fatalf("Failed to load token from path %s: %v", o.tokenPath, err)
	}

	ReportOutputPath = BuildReportOutputPath(o)

	for _, ep := range o.endpoint.Strings() {
		_, err = url.ParseRequestURI(ep)
		if err != nil {
			log.Fatalf("Invalid --endpoint URL %q: %v.", ep, err)
		}
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: string(secret.GetSecret(o.tokenPath))},
	)
	tc := oauth2.NewClient(ctx, ts)

	ghClient := github.NewClient(tc)

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create new storage client: %v.\n", err)
	}

	reportBaseDataOptions := flakefinder.NewReportBaseDataOptions(o.prBaseBranch, o.today, o.merged, o.org, o.repo, o.skipBeforeStartOfReport)
	reportBaseDataOptions.SetPeriodicJobDirRegex(o.periodicJobDirRegex)

	reportBaseData := flakefinder.GetReportBaseData(ctx, ghClient, storageClient, reportBaseDataOptions)

	err = WriteReportToBucket(ctx, storageClient, o.merged, o.org, o.repo, o.isDryRun, reportBaseData)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to write report: %v", err))
		return
	}

	printIndexPageToStdOut := o.isDryRun
	err = CreateReportIndex(ctx, storageClient, o.org, o.repo, printIndexPageToStdOut)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create report index page: %v", err))
		return
	}
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

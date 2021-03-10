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
 * Copyright 2021 Red Hat, Inc.
 *
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/go-github/v28/github"
	"golang.org/x/oauth2"
	"k8s.io/test-infra/prow/config/secret"
	"k8s.io/test-infra/prow/flagutil"
	prowgithub "k8s.io/test-infra/prow/github"

	"kubevirt.io/project-infra/robots/pkg/flakefinder"
)

const (
	DefaultIssueLabels  = "triage/build-watcher,kind/bug"
	DefaultIssueTitlePrefix  = "[flaky ci]"
	DeckPRLogURLPattern = "https://prow.apps.ovirt.org/view/gcs/kubevirt-prow/pr-logs/pull/kubevirt_kubevirt/%d/%s/%d"
)

func flagOptions() options {
	o := options{
		endpoint: flagutil.NewStrings("https://api.github.com"),
	}
	flag.BoolVar(&o.isDryRun, "dry-run", true, "whether data modification should be performed or just printed to stdout, i.e. creation of issues etc.")
	flag.Var(&o.endpoint, "endpoint", "GitHub's API endpoint")
	flag.StringVar(&o.token, "token", "", "Path to github token")
	flag.DurationVar(&o.merged, "merged", 24*7*time.Hour, "Filter to issues merged in the time window")
	flag.StringVar(&o.prBaseBranch, "pr_base_branch", PRBaseBranchDefault, "Base branch for the PRs")
	flag.StringVar(&o.org, "org", Org, "GitHub org name")
	flag.StringVar(&o.repo, "repo", Repo, "GitHub org name")
	flag.StringVar(&o.createFlakeIssuesLabels, "flake-issue-labels", DefaultIssueLabels, "Labels to attach to created issues")
	flag.IntVar(&o.createFlakeIssuesThreshold, "flake-issue-threshold", 0, "Maximum number of issues to create, 0 for all")
	flag.IntVar(&o.suspectedClusterFailureThreshold, "suspected-cluster-failure-threshold", 50, "Minimum number of test failures in one job to aggregate all test failures into one issue, 0 for never")
	flag.Parse()
	return o
}

type options struct {
	isDryRun                         bool
	endpoint                         flagutil.Strings
	token                            string
	graphqlEndpoint                  string
	merged                           time.Duration
	prBaseBranch                     string
	org                              string
	repo                             string
	createFlakeIssuesLabels          string
	createFlakeIssuesThreshold       int
	suspectedClusterFailureThreshold int
}

const PRBaseBranchDefault = "master"
const Org = "kubevirt"
const Repo = "kubevirt"

var PRBaseBranch string
var ctx context.Context

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	o := flagOptions()

	if o.token == "" {
		log.Fatal("empty --token")
	}

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

	ctx = context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: string(secretAgent.GetSecret(o.token))},
	)
	tc := oauth2.NewClient(ctx, ts)

	ghClient := github.NewClient(tc)

	var sa secret.Agent
	if err := sa.Start([]string{o.token}); err != nil {
		log.Fatalf("failed to start secrets agent: %w", err)
	}
	pghClient := prowgithub.NewClient(sa.GetTokenGenerator(o.token), sa.Censor, prowgithub.DefaultGraphQLEndpoint, prowgithub.DefaultAPIEndpoint)

	labels, err := pghClient.GetRepoLabels(o.org, o.repo)
	if err != nil {
		log.Fatalf("Failed to fetch labels for %s/%s: %v.\n", o.org, o.repo, err)
	}

	flakeIssuesLabels, err := GetFlakeIssuesLabels(o.createFlakeIssuesLabels, labels, o.org, o.repo)
	if err != nil {
		log.Fatalf("Failed to get flake issue labels: %v.\n", err)
	}

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create new storage client: %v.\n", err)
	}

	reportBaseData := flakefinder.GetReportBaseData(ctx, ghClient, storageClient, flakefinder.NewReportBaseDataOptions(PRBaseBranch, false, o.merged, o.org, o.repo, false))

	reportData := flakefinder.CreateFlakeReportData(reportBaseData.JobResults, reportBaseData.PRNumbers, reportBaseData.EndOfReport, o.org, o.repo, reportBaseData.StartOfReport)

	clusterFailureBuildNumbers, err := CreateClusterFailureIssues(reportData, o.suspectedClusterFailureThreshold, flakeIssuesLabels, pghClient, o.isDryRun)
	if err != nil {
		log.Fatalf("Failed to create cluster failure issues: %v.\n", err)
	}
	fmt.Printf("clusterFailureBuildNumbers: %+v", clusterFailureBuildNumbers)

	// TODO: create issues for flaky tests
	CreateFlakyTestIssues(reportData, clusterFailureBuildNumbers, flakeIssuesLabels, pghClient, o.isDryRun)

}


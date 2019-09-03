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
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"k8s.io/test-infra/prow/config/secret"
	"k8s.io/test-infra/prow/flagutil"
)

func flagOptions() options {
	o := options{
		endpoint: flagutil.NewStrings("https://api.github.com"),
	}
	flag.IntVar(&o.ceiling, "ceiling", 100, "Maximum number of issues to modify, 0 for infinite")
	flag.DurationVar(&o.merged, "merged", 24*7*time.Hour, "Filter to issues merged in the time window")
	flag.Var(&o.endpoint, "endpoint", "GitHub's API endpoint")
	flag.StringVar(&o.token, "token", "", "Path to github token")
	flag.Parse()
	return o
}

type options struct {
	ceiling         int
	endpoint        flagutil.Strings
	token           string
	graphqlEndpoint string
	merged          time.Duration
}

type client interface {
	FindIssues(query, sort string, asc bool) ([]github.Issue, error)
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
}

const BucketName = "kubevirt-prow"
const ReportsPath = "reports/flakefinder"
const ReportFilePrefix = "flakefinder-"
const MaxNumberOfReportsToLinkTo = 50

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	o := flagOptions()

	if o.token == "" {
		log.Fatal("empty --token")
	}

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

	// we are fetching reports from start of day to avoid working against a moving target.
	// In general a user would expect to find all pull requests of the previous day in a 24h report, regardless of
	// when the report has been run at the current day, which, depending on time of day when the report had been run,
	// would not always be the case.
	// Consider i.e. if the report is run late in the afternoon the user might wonder why the PR merged in the morning
	// the day before was not included.
	startOfReport := time.Now().Add(-o.merged)
	startOfDay := startOfReport.Format("2006-01-02") + "T00:00:00Z"
	logrus.Infof("Fetching Prs starting from %v", startOfDay)
	startOfReport, err = time.Parse(time.RFC3339, startOfDay)
	if err != nil {
		log.Fatalf("Failed to parse time %+v: %+v", startOfDay, err)
	}

	prs := []*github.PullRequest{}
	for nextPage := 1; nextPage > 0; {
		pullRequests, response, err := c.PullRequests.List(ctx, "kubevirt", "kubevirt", &github.PullRequestListOptions{
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
			logrus.Infof("Adding PR %v '%v' (updated at %s)", *pr.Number, *pr.Title, pr.UpdatedAt.Format(time.RFC3339))
			prs = append(prs, pr)
		}
	}
	logrus.Infof("%d pull requests found.", len(prs))

	client, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatalf("Failed to create new storage client: %v.\n", err)
	}
	reports := []*Result{}
	for _, pr := range prs {
		r, err := FindUnitTestFiles(ctx, client, BucketName, "kubevirt/kubevirt", pr)
		if err != nil {
			log.Printf("failed to load JUnit file for %v: %v", pr.Number, err)
		}
		reports = append(reports, r...)
	}

	err = WriteReportToBucket(ctx, client, reports, o.merged)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to write report: %v", err))
		return
	}

	err = CreateReportIndex(ctx, client)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create report index page: %v", err))
		return
	}

}

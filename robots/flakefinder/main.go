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
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"k8s.io/test-infra/prow/config/secret"
	"k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/github"
)

func flagOptions() options {
	o := options{
		endpoint: flagutil.NewStrings("https://api.github.com"),
	}
	flag.IntVar(&o.ceiling, "ceiling", 100, "Maximum number of issues to modify, 0 for infinite")
	flag.DurationVar(&o.merged, "merged", 24*7*time.Hour, "Filter to issues merged in the time window")
	flag.Var(&o.endpoint, "endpoint", "GitHub's API endpoint")
	flag.StringVar(&o.token, "token", "", "Path to github token")
	flag.StringVar(&o.graphqlEndpoint, "graphql-endpoint", github.DefaultGraphQLEndpoint, "GitHub's GraphQL API Endpoint")
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

	var c client = github.NewClient(secretAgent.GetTokenGenerator(o.token), o.graphqlEndpoint, o.endpoint.Strings()...)
	query := MakeQuery("repo:kubevirt/kubevirt is:merged is:pr", o.merged, time.Now())
	issues, err := c.FindIssues(query, "", false)
	if err != nil {
		log.Fatalf("Failed run: %v", err)
	}

	prs := []*github.PullRequest{}
	for _, issue := range issues {
		pr, err := c.GetPullRequest("kubevirt", "kubevirt", issue.Number)
		if err != nil {
			log.Fatalf("Failed to fetch PR: %v.\n", err)
		}
		prs = append(prs, pr)
	}

	ctx := context.Background()
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

func MakeQuery(query string, minMerged time.Duration, till time.Time) string {
	parts := []string{query}
	if minMerged != 0 {
		latest := till.Add(-minMerged)
		parts = append(parts, "merged:>="+latest.Format(time.RFC3339))
	}
	return strings.Join(parts, " ")
}

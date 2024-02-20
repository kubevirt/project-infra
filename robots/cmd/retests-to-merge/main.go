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
	"context"
	"flag"
	"os"
	"strings"

	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"kubevirt.io/project-infra/external-plugins/referee/ghgraphql"
)

func main() {
	var tokenPath string
	flag.StringVar(&tokenPath, "github-token-path", "", "the path to the GitHub token to use")
	flag.Parse()

	token, err := os.ReadFile(tokenPath)
	if err != nil {
		logrus.Fatalf("failed to use github token path %s: %v", tokenPath, err)
	}
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: strings.TrimSpace(string(token))},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	var gitHubClient *githubv4.Client
	gitHubClient = githubv4.NewClient(httpClient)

	org := "kubevirt"
	repo := "kubevirt"

	gitHubGraphQLClient := ghgraphql.NewClient(gitHubClient)

	pullRequests, err := gitHubGraphQLClient.FetchOpenApprovedAndLGTMedPRs(org, repo)
	if err != nil {
		logrus.Fatalf("failed to fetch pull requests for %s/%s: %v", org, repo, err)
	}

	for _, pr := range pullRequests.PRs {
		prTimeLineForLastCommit, err := gitHubGraphQLClient.FetchPRTimeLineForLastCommit(org, repo, pr.Number)
		if err != nil {
			logrus.Fatalf("failed to fetch number of retest comments for pr %s/%s#%d: %v", org, repo, pr.Number, err)
		}
		if prTimeLineForLastCommit.NumberOfRetestComments < 5 {
			continue
		}
		logrus.Infof("https://github.com/%s/%s/pull/%d - pr %q has %d retests", org, repo, pr.Number, pr.Title, prTimeLineForLastCommit.NumberOfRetestComments)
		labels, err := gitHubGraphQLClient.FetchPRLabels(org, repo, pr.Number)
		if err != nil {
			logrus.Fatalf("failed to fetch number of retest comments for pr %s/%s#%d: %v", org, repo, pr.Number, err)
		}
		logrus.Infof("labels for PR: %+v", labels)
	}
}

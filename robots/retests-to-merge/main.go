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
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"

	"kubevirt.io/project-infra/external-plugins/referee/ghgraphql"
)

type result struct {
	ghgraphql.PullRequest
	ghgraphql.PRTimelineForLastCommit
	ghgraphql.PRLabels
}

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
}

func main() {
	var tokenPath, org, repo string
	flag.StringVar(&tokenPath, "github-token-path", "", "the path to the GitHub token to use")
	flag.StringVar(&org, "org", "kubevirt", "the path to the GitHub token to use")
	flag.StringVar(&repo, "repo", "kubevirt", "the path to the GitHub token to use")
	flag.Parse()

	log := logrus.WithField("robot", "retests-to-merge").WithField("repo", fmt.Sprintf("%s/%s", org, repo))

	token, err := os.ReadFile(tokenPath)
	if err != nil {
		log.Fatalf("failed to use github token path %s: %v", tokenPath, err)
	}
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: strings.TrimSpace(string(token))},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	gitHubClient := githubv4.NewClient(httpClient)

	gitHubGraphQLClient := ghgraphql.NewClient(gitHubClient)

	pullRequests, err := gitHubGraphQLClient.FetchOpenApprovedAndLGTMedPRs(org, repo)
	if err != nil {
		log.Fatalf("failed to fetch pull requests for %s/%s: %v", org, repo, err)
	}

	numRetestsToPRNumsToResults := make(map[int]map[int]result)
	numRetests := make(map[int]struct{})
	log.Infof("checking %d PRs for retests", len(pullRequests.PRs))
	for _, pr := range pullRequests.PRs {
		prLog := log.WithField("pr_url", fmt.Sprintf("https://github.com/%s/%s/pull/%d", org, repo, pr.Number)).WithField("pr_title", pr.Title)
		prTimeLineForLastCommit, err := gitHubGraphQLClient.FetchPRTimeLineForLastCommit(org, repo, pr.Number)
		if err != nil {
			prLog.Fatalf("failed to fetch number of retest comments for pr %s/%s#%d: %v", org, repo, pr.Number, err)
		}
		if prTimeLineForLastCommit.NumberOfRetestComments <= 0 {
			continue
		}
		numRetests[prTimeLineForLastCommit.NumberOfRetestComments] = struct{}{}
		if _, ok := numRetestsToPRNumsToResults[prTimeLineForLastCommit.NumberOfRetestComments]; !ok {
			numRetestsToPRNumsToResults[prTimeLineForLastCommit.NumberOfRetestComments] = make(map[int]result)
		}
		labels, err := gitHubGraphQLClient.FetchPRLabels(org, repo, pr.Number)
		if err != nil {
			prLog.Fatalf("failed to fetch number of retest comments: %v", err)
		}
		numRetestsToPRNumsToResults[prTimeLineForLastCommit.NumberOfRetestComments][pr.Number] = result{
			PullRequest:             pr,
			PRTimelineForLastCommit: prTimeLineForLastCommit,
			PRLabels:                labels,
		}
	}
	if len(numRetests) == 0 {
		log.Infof("no excessive retests found - hooray")
	}
	numRetestsSlice := make([]int, len(numRetests))
	for numRetest := range numRetests {
		numRetestsSlice = append(numRetestsSlice, numRetest)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(numRetestsSlice)))
	log.Infof("max number of retests encountered: %d", numRetestsSlice[0])
	for _, numRetest := range numRetestsSlice {
		numRetestLog := log.WithField("num_retests", numRetest)
		for _, results := range numRetestsToPRNumsToResults[numRetest] {
			numRetestLog.WithField("pr_url", fmt.Sprintf("https://github.com/%s/%s/pull/%d", org, repo, results.PullRequest.Number)).WithField("pr_title", results.PullRequest.Title).Infof("labels: %v", results.PRLabels)
		}
	}
}

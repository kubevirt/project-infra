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
	"fmt"
	"k8s.io/test-infra/prow/github"
	"log"
	"sort"
	"strings"
)

func GetFlakeIssuesLabels(createFlakeIssuesLabels string, labels []github.Label, org, repo string) (issueLabels []github.Label, err error) {
	configuredIssueLabels := strings.Split(createFlakeIssuesLabels, ",")
	sort.Strings(configuredIssueLabels)
	for _, label := range labels {
		for _, configuredLabel := range configuredIssueLabels {
			if configuredLabel == label.Name {
				issueLabels = append(issueLabels, label)
				index := sort.SearchStrings(configuredIssueLabels, configuredLabel)
				configuredIssueLabels = append(configuredIssueLabels[:index], configuredIssueLabels[index+1:]...)
				break
			}
		}
	}
	if len(configuredIssueLabels) > 0 {
		return nil, fmt.Errorf("labels %+v not found for %s/%s.\n", configuredIssueLabels, org, repo)
	}
	return
}

func CreateProwJobURL(failingPR int, failingTestLane string, clusterFailureBuildNumber int, org string, repo string) string {
	return fmt.Sprintf(DeckPRLogURLPattern, org, repo, failingPR, failingTestLane, clusterFailureBuildNumber)
}

func CreateIssues(org, repo string, labels []github.Label, issues []github.Issue, client github.Client, dryRun bool) error {
	labelNames := extractLabelNames(labels)
	labelSearch := createSearchByLabelsExpression(labels)
	for _, issue := range issues {
		findIssues, err := client.FindIssues(fmt.Sprintf(" org:%s repo:%s %s \"%s\"", org, repo, labelSearch, issue.Title), "updated-desc", false)
		if err != nil {
			return err
		}
		if len(findIssues) > 0 {
			log.Printf("Issues found: %+v", findIssues)
			latestExistingIssue := findIssues[0]
			if latestExistingIssue.State == "closed" {
				log.Printf("Reopen issue: %+v", latestExistingIssue)
				if !dryRun {
					err := client.ReopenIssue(org, repo, latestExistingIssue.ID)
					if err != nil {
						return err
					}
				}
			}
			log.Printf("Create comment on issue %d: %s", latestExistingIssue.ID, issue.Body)
			if !dryRun {
				err := client.CreateComment(org, repo, latestExistingIssue.ID, issue.Body)
				if err != nil {
					return err
				}
			}
			continue
		}

		var createdIssue int
		log.Printf("Create issue: %+v", issue)
		if !dryRun {
			createdIssue, err = client.CreateIssue(org, repo, issue.Title, issue.Body, 0, labelNames, nil)
			if err != nil {
				return err
			}
		}
		log.Printf("Created issue %d %+v", createdIssue, issue)
	}
	return nil
}


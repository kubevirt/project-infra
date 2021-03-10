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
	. "kubevirt.io/project-infra/robots/pkg/flakefinder"
	"log"
	"strings"
)

func CreateClusterFailureIssues(reportData Params, suspectedClusterFailureThreshold int, labels []github.Label, client github.Client, dryRun bool) (clusterFailureBuildNumbers []int, err error) {
	var issues []github.Issue
	issues, clusterFailureBuildNumbers = NewClusterFailureIssues(reportData, suspectedClusterFailureThreshold, labels)

	labelNames := extractLabelNames(labels)

	for _, issue := range issues {
		labelSearch := createSearchByLabelsExpression(labels)
		findIssues, err := client.FindIssues(fmt.Sprintf("%s \"%s\"", labelSearch, issue.Title), "updated-desc", false)
		if err != nil {
			return nil, err
		}
		if len(findIssues) > 0 {
			log.Printf("Issues found: %+v", findIssues)
			latestExistingIssue := findIssues[0]
			if latestExistingIssue.State == "closed" {
				log.Printf("Reopen issue: %+v", latestExistingIssue)
				if !dryRun {
					err := client.ReopenIssue(reportData.Org, reportData.Repo, latestExistingIssue.ID)
					if err != nil {
						return nil, err
					}
				}
			}
			log.Printf("Create comment on issue %d: %s", latestExistingIssue.ID, issue.Body)
			if !dryRun {
				err := client.CreateComment(reportData.Org, reportData.Repo, latestExistingIssue.ID, issue.Body)
				if err != nil {
					return nil, err
				}
			}
			continue
		}

		var createdIssue int
		log.Printf("Create issue: %+v", issue)
		if !dryRun {
			createdIssue, err = client.CreateIssue(reportData.Org, reportData.Repo, issue.Title, issue.Body, 0, labelNames, nil)
			if err != nil {
				return nil, err
			}
		}
		log.Printf("Created issue %d %+v", createdIssue, issue)
	}
	return clusterFailureBuildNumbers, nil
}

func NewClusterFailureIssues(reportData Params, suspectedClusterFailureThreshold int, labels []github.Label) (issues []github.Issue, clusterFailureBuildNumbers []int) {
	for buildNumber, failure := range reportData.FailuresForJobs {
		if failure.Failures < suspectedClusterFailureThreshold {
			continue
		}
		clusterFailureBuildNumbers = append(clusterFailureBuildNumbers, buildNumber)
		clusterFailureIssue := github.Issue{
			Title: fmt.Sprintf("[flaky ci] %s temporary cluster failure", failure.Job),
			Body: fmt.Sprintf("Test lane failed on %d tests: %s", failure.Failures, CreateProwJobURL(failure.PR, failure.Job, failure.BuildNumber)),
			Labels: labels,
		}
		issues = append(issues, clusterFailureIssue)
	}
	return
}

func extractLabelNames(labels []github.Label) []string {
	var result []string
	for _, label := range labels {
		result = append(result, label.Name)
	}
	return result
}

func createSearchByLabelsExpression(labels []github.Label) string {
	var parts []string
	for _, label := range labels {
		parts = append(parts, fmt.Sprintf("label:%s", label.Name))
	}
	return strings.Join(parts, " ")
}

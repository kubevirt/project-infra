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
	"time"
)

func CreateClusterFailureIssues(reportData Params, suspectedClusterFailureThreshold int, labels []github.Label, client github.Client, dryRun bool, createIssueThreshold int, skipExistingIssuesChangedLately time.Duration) (clusterFailureBuildNumbers []int, err error) {
	if suspectedClusterFailureThreshold <= 0 {
		log.Print("Skipping cluster failure creation\n")
		return nil, nil
	}

	var issues []github.Issue
	issues, clusterFailureBuildNumbers = NewClusterFailureIssues(reportData, suspectedClusterFailureThreshold, labels)

	if createIssueThreshold > 0 && len(issues) > createIssueThreshold {
		log.Printf("Create issue threshold reached, skipping creation of %d issue(s):\n%v", len(issues)-createIssueThreshold, issues[createIssueThreshold:])
		issues = issues[:createIssueThreshold]
	}

	err = CreateIssues(reportData.Org, reportData.Repo, labels, issues, client, dryRun, skipExistingIssuesChangedLately)
	if err != nil {
		return nil, err
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
			Title:  fmt.Sprintf("[flaky ci] %s temporary cluster failure", failure.Job),
			Body:   fmt.Sprintf("Test lane failed on %d tests: %s", failure.Failures, CreateProwJobURL(failure.PR, failure.Job, failure.BuildNumber, reportData.Org, reportData.Repo)),
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

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
	"kubevirt.io/project-infra/robots/pkg/flakefinder"
	"log"
	"sort"
)

func CreateFlakyTestIssues(reportData flakefinder.Params, clusterFailureBuildNumbers []int, flakeIssuesLabels []github.Label, pghClient github.Client, isDryRun bool, createIssuesThreshold int) error {
	flakyTestIssues := NewFlakyTestIssues(reportData, clusterFailureBuildNumbers, flakeIssuesLabels)

	if createIssuesThreshold > 0 && len(flakyTestIssues) > createIssuesThreshold {
		log.Printf("Create issue threshold reached, skipping creation of %d issue(s):\n%v", len(flakyTestIssues)-createIssuesThreshold, flakyTestIssues[createIssuesThreshold:])
		flakyTestIssues = flakyTestIssues[:createIssuesThreshold]
	}

	err := CreateIssues(reportData.Org, reportData.Repo, flakeIssuesLabels, flakyTestIssues, pghClient, isDryRun)
	if err != nil {
		return fmt.Errorf("failed to create flaky test issues: %+v", err)
	}
	return nil
}

func NewFlakyTestIssues(reportData flakefinder.Params, clusterFailureBuildNumbers []int, labels []github.Label) (flakyTestIssues []github.Issue) {
	sort.Ints(clusterFailureBuildNumbers)
	for testName, laneData := range reportData.Data {
		issueBody := ""
		var flakyTestIssue github.Issue
		for laneName, laneDetails := range laneData {
			if laneDetails.Failed <= 0 {
				continue
			}
			issueBodyJobLanes := ""
			for _, job := range laneDetails.Jobs {
				if index := sort.SearchInts(clusterFailureBuildNumbers, job.BuildNumber); index < len(clusterFailureBuildNumbers) && clusterFailureBuildNumbers[index] == job.BuildNumber {
					continue
				}
				if issueBodyJobLanes == "" {
					issueBodyJobLanes = fmt.Sprintf("Lane %s failed on job runs:", laneName)
				}
				issueBodyJobLanes += fmt.Sprintf("\n* Prow job id %d: %s", job.BuildNumber, CreateProwJobURL(job.PR, job.Job, job.BuildNumber, reportData.Org, reportData.Repo))
			}
			if issueBodyJobLanes != "" {
				issueBody += issueBodyJobLanes
			}
		}
		if issueBody != "" {
			flakyTestIssue = github.Issue{
				Title:  fmt.Sprintf("%s%s", DefaultIssueTitlePrefix, testName),
				Body:   issueBody,
				Labels: labels,
			}
			flakyTestIssues = append(flakyTestIssues, flakyTestIssue)
		}
	}
	return
}

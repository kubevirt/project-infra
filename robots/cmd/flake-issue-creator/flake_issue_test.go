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

package main_test

import (
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	gh "k8s.io/test-infra/prow/github"
	"strings"

	. "kubevirt.io/project-infra/robots/cmd/flake-issue-creator"
	. "kubevirt.io/project-infra/robots/pkg/flakefinder"
)

var _ = Describe("flake_issue.go", func() {

	jobTestFailures := 10
	jobBuildNumber := 37
	clusterFailureJobBuildNumber := 666
	clusterFailureBuildNumbers := []int{clusterFailureJobBuildNumber}
	failingTestLane := "pull-whatever"
	failingPR := 17
	data := map[string]map[string]*Details{
		"[rfe_id:1234][crit:high][owner:@sig-compute][test_id:2345]test case description": {
			failingTestLane: &Details{Failed: 3, Jobs: []*Job{
				{BuildNumber: jobBuildNumber, Severity: "hard", PR: failingPR, Job: failingTestLane},
			}},
		},
		"[rfe_id:1234][crit:high][owner:@sig-compute][test_id:3456]test case description": {
			failingTestLane: &Details{Failed: 3, Jobs: []*Job{
				{BuildNumber: jobBuildNumber, Severity: "hard", PR: failingPR, Job: failingTestLane},
			}},
		},
		"[rfe_id:1234][crit:high][owner:@sig-compute][test_id:4567]test case description": {
			failingTestLane: &Details{Failed: 4, Jobs: []*Job{
				{BuildNumber: jobBuildNumber, Severity: "hard", PR: failingPR, Job: failingTestLane},
			}},
		},
		"[rfe_id:1234][crit:high][owner:@sig-compute][test_id:5678]test case description": {
			failingTestLane: &Details{Failed: 5, Jobs: []*Job{
				{BuildNumber: clusterFailureJobBuildNumber, Severity: "hard", PR: failingPR, Job: failingTestLane},
			}},
		},
		"[rfe_id:1234][crit:high][owner:@sig-compute][test_id:6789]test case description": {
			failingTestLane: &Details{Failed: 0, Jobs: []*Job{
				{BuildNumber: jobBuildNumber, Severity: "hard", PR: failingPR, Job: failingTestLane},
			}},
		},
	}
	jobFailures := JobFailures{BuildNumber: jobBuildNumber, PR: failingPR, Job: failingTestLane, Failures: jobTestFailures}
	params := Params{
		Org:             "kubevirt",
		Repo:            "kubevirt",
		Data:            data,
		FailuresForJobs: map[int]*JobFailures{jobBuildNumber: &jobFailures},
	}

	buildWatcher := "triage/build-watcher"
	typeBug := "type/bug"
	issueLabels := []gh.Label{
		{Name: buildWatcher},
		{Name: typeBug},
	}

	When("extracting new flaky test issues", func() {

		It("returns new flake test issues", func() {
			issues := NewFlakyTestIssues(params, clusterFailureBuildNumbers, issueLabels)
			gomega.Expect(issues).To(gomega.Not(gomega.BeNil()))
			gomega.Expect(issues).To(gomega.HaveLen(3))
		})

		It("flake test issues have test title", func() {
			issues := NewFlakyTestIssues(params, clusterFailureBuildNumbers, issueLabels)
			gomega.Expect(
				func() string {
					var result []string
					for _, issue := range issues {
						result = append(result, issue.Title)
					}
					return strings.Join(result, ",")
				}(),
			).To(gomega.ContainSubstring("test_id:2345"))
		})

		It("flake test issues have test body with lane name", func() {
			issues := NewFlakyTestIssues(params, clusterFailureBuildNumbers, issueLabels)
			gomega.Expect(issues[0].Body).To(gomega.ContainSubstring(failingTestLane))
		})

		It("flake test issues have test body with lane name", func() {
			issues := NewFlakyTestIssues(params, clusterFailureBuildNumbers, issueLabels)
			gomega.Expect(issues[0].Body).To(gomega.ContainSubstring(failingTestLane))
		})

		It("flake test issues have test body with URL", func() {
			issues := NewFlakyTestIssues(params, clusterFailureBuildNumbers, issueLabels)
			gomega.Expect(issues[0].Body).To(gomega.ContainSubstring("http"))
		})

		PIt("uses org and repo when creating issues", func() {
			Fail("TODO") // TODO
		})

	})

	When("creating flaky test issues", func() {

		PIt("stops after limit of creation has been reached", func() {
			Fail("TODO") // TODO
		})

	})

})

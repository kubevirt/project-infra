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
	"fmt"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	gh "k8s.io/test-infra/prow/github"

	. "kubevirt.io/project-infra/robots/cmd/flake-issue-creator"
	. "kubevirt.io/project-infra/robots/pkg/flakefinder"
)

var _ = Describe("cluster_failure.go", func() {

	clusterFailures := 10

	clusterFailureBuildNumber := 37
	failingTestLane := "pull-whatever"
	failingPR := 17
	data := map[string]map[string]*Details{
		"[rfe_id:1234][crit:high][owner:@sig-compute][test_id:2345]test case description": {
			failingTestLane: &Details{Failed: 3, Jobs: []*Job{
				{BuildNumber: clusterFailureBuildNumber, Severity: "hard", PR: failingPR, Job: failingTestLane},
			}},
		},
		"[rfe_id:1234][crit:high][owner:@sig-compute][test_id:3456]test case description": {
			failingTestLane: &Details{Failed: 3, Jobs: []*Job{
				{BuildNumber: clusterFailureBuildNumber, Severity: "hard", PR: failingPR, Job: failingTestLane},
			}},
		},
		"[rfe_id:1234][crit:high][owner:@sig-compute][test_id:4567]test case description": {
			failingTestLane: &Details{Failed: 4, Jobs: []*Job{
				{BuildNumber: clusterFailureBuildNumber, Severity: "hard", PR: failingPR, Job: failingTestLane},
			}},
		},
	}
	jobFailures := JobFailures{BuildNumber: clusterFailureBuildNumber, PR: failingPR, Job: failingTestLane, Failures: clusterFailures}
	params := Params{
		Org:             "kubevirt",
		Repo:            "kubevirt",
		Data:            data,
		FailuresForJobs: map[int]*JobFailures{clusterFailureBuildNumber: &jobFailures},
	}

	buildWatcher := "triage/build-watcher"
	typeBug := "type/bug"
	issueLabels := []gh.Label{
		{Name: buildWatcher},
		{Name: typeBug},
	}

	When("extracting cluster failure issues", func() {

		It("returns nil on empty values", func() {
			issues, clusterFailureBuildNumbers := NewClusterFailureIssues(Params{}, clusterFailures, issueLabels)
			gomega.Expect(issues).To(gomega.BeNil())
			gomega.Expect(clusterFailureBuildNumbers).To(gomega.BeNil())
		})

		It("creates cluster failure on failures within threshold", func() {
			issues, clusterFailureBuildNumbers := NewClusterFailureIssues(params, clusterFailures, issueLabels)
			gomega.Expect(issues).To(gomega.Not(gomega.BeNil()))
			gomega.Expect(clusterFailureBuildNumbers).To(gomega.BeEquivalentTo([]int{clusterFailureBuildNumber}))
		})

		It("does not create cluster failure on failures below threshold", func() {
			issues, clusterFailureBuildNumbers := NewClusterFailureIssues(params, 11, issueLabels)
			gomega.Expect(issues).To(gomega.BeNil())
			gomega.Expect(clusterFailureBuildNumbers).To(gomega.BeNil())
		})

		It("creates an issue with a title that describes the failed lane", func() {
			issues, _ := NewClusterFailureIssues(params, clusterFailures, issueLabels)
			gomega.Expect(issues[0].Title).To(gomega.ContainSubstring(fmt.Sprintf("[flaky ci] %s temporary cluster failure", failingTestLane)))
		})

		It("creates an issue with links to the failed job", func() {
			issues, _ := NewClusterFailureIssues(params, clusterFailures, issueLabels)
			gomega.Expect(issues[0].Body).To(gomega.ContainSubstring(fmt.Sprintf("Test lane failed on %d tests: %s", clusterFailures, CreateProwJobURL(failingPR, failingTestLane, clusterFailureBuildNumber))))
		})

		It("creates an issue with labels", func() {
			issues, _ := NewClusterFailureIssues(params, clusterFailures, issueLabels)
			labels := func() []string {
				var result []string
				for _, label := range issues[0].Labels {
					result = append(result, label.Name)
				}
				return result
			}()
			gomega.Expect(labels).To(gomega.BeEquivalentTo([]string{buildWatcher, typeBug}))
		})

		PIt("uses org and repo when creating issues", func() {
			Fail("TODO") // TODO
		})

	})

	When("creating cluster failure issues", func() {

		PIt("stops after limit of creation has been reached", func() {
			Fail("TODO") // TODO
		})

	})

})

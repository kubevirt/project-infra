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
	"kubevirt.io/project-infra/robots/pkg/gomock/matchers"

	. "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	gh "k8s.io/test-infra/prow/github"

	. "kubevirt.io/project-infra/robots/cmd/flake-issue-creator"
	. "kubevirt.io/project-infra/robots/pkg/flakefinder"
	. "kubevirt.io/project-infra/robots/pkg/mock/prow/github"
)

var _ = Describe("cluster_failure.go", func() {

	suspectedClusterFailureThreshold := 10
	clusterFailures := 10

	clusterFailureBuildNumber := 37
	failingTestLane := "pull-whatever"
	failingPR := 17
	buildWatcher := "triage/build-watcher"
	typeBug := "type/bug"
	issueLabels := []gh.Label{
		{Name: buildWatcher},
		{Name: typeBug},
	}

	When("extracting cluster failure issues", func() {

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

		params := Params{
			Org:  "kubevirt",
			Repo: "kubevirt",
			Data: data,
			FailuresForJobs: map[int]*JobFailures{
				clusterFailureBuildNumber: {
					BuildNumber: clusterFailureBuildNumber,
					PR:          failingPR,
					Job:         failingTestLane,
					Failures:    clusterFailures,
				},
			},
		}

		It("returns nil on empty values", func() {
			issues, clusterFailureBuildNumbers := NewClusterFailureIssues(Params{}, suspectedClusterFailureThreshold, issueLabels)
			gomega.Expect(issues).To(gomega.BeNil())
			gomega.Expect(clusterFailureBuildNumbers).To(gomega.BeNil())
		})

		It("creates cluster failure on failures within threshold", func() {
			issues, clusterFailureBuildNumbers := NewClusterFailureIssues(params, suspectedClusterFailureThreshold, issueLabels)
			gomega.Expect(issues).ToNot(gomega.BeNil())
			gomega.Expect(clusterFailureBuildNumbers).To(gomega.BeEquivalentTo([]int{clusterFailureBuildNumber}))
		})

		It("does not create cluster failure on failures below threshold", func() {
			issues, clusterFailureBuildNumbers := NewClusterFailureIssues(params, 11, issueLabels)
			gomega.Expect(issues).To(gomega.BeNil())
			gomega.Expect(clusterFailureBuildNumbers).To(gomega.BeNil())
		})

		It("creates an issue with a title that describes the failed lane", func() {
			issues, _ := NewClusterFailureIssues(params, suspectedClusterFailureThreshold, issueLabels)
			gomega.Expect(issues[0].Title).To(gomega.ContainSubstring(fmt.Sprintf("[flaky ci] %s temporary cluster failure", failingTestLane)))
		})

		It("creates an issue with links to the failed job", func() {
			issues, _ := NewClusterFailureIssues(params, suspectedClusterFailureThreshold, issueLabels)
			gomega.Expect(issues[0].Body).To(gomega.ContainSubstring(fmt.Sprintf("Test lane failed on %d tests: %s", suspectedClusterFailureThreshold, CreateProwJobURL(failingPR, failingTestLane, clusterFailureBuildNumber, params.Org, params.Repo))))
		})

		It("creates an issue with labels", func() {
			issues, _ := NewClusterFailureIssues(params, suspectedClusterFailureThreshold, issueLabels)
			labels := func() []string {
				var result []string
				for _, label := range issues[0].Labels {
					result = append(result, label.Name)
				}
				return result
			}()
			gomega.Expect(labels).To(gomega.BeEquivalentTo([]string{buildWatcher, typeBug}))
		})
	})

	When("extracting cluster failure issues for another org and repo", func() {

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

		params := Params{
			Org:             "myorg",
			Repo:            "myrepo",
			Data:            data,
			FailuresForJobs: map[int]*JobFailures{clusterFailureBuildNumber: &JobFailures{BuildNumber: clusterFailureBuildNumber, PR: failingPR, Job: failingTestLane, Failures: suspectedClusterFailureThreshold}},
		}

		It("uses org and repo when creating issues", func() {
			issues, _ := NewClusterFailureIssues(params, suspectedClusterFailureThreshold, issueLabels)
			fmt.Println(issues)
			gomega.Expect(issues[0].Title).ToNot(gomega.ContainSubstring("kubevirt"))
			gomega.Expect(issues[0].Body).To(gomega.ContainSubstring("myorg_myrepo"))
		})

	})

	When("creating cluster failure issues", func() {

		anotherClusterFailureBuildNumber := 38
		anotherClusterFailTestLane := "pull-another-cluster-fail-lane"

		data := map[string]map[string]*Details{
			"[rfe_id:1234][crit:high][owner:@sig-compute][test_id:2345]test case description": {
				failingTestLane: &Details{Failed: 3, Jobs: []*Job{
					{BuildNumber: clusterFailureBuildNumber, Severity: "hard", PR: failingPR, Job: failingTestLane},
				}},
				anotherClusterFailTestLane: &Details{Failed: 3, Jobs: []*Job{
					{BuildNumber: anotherClusterFailureBuildNumber, Severity: "hard", PR: failingPR, Job: anotherClusterFailTestLane},
				}},
			},
			"[rfe_id:1234][crit:high][owner:@sig-compute][test_id:3456]test case description": {
				failingTestLane: &Details{Failed: 3, Jobs: []*Job{
					{BuildNumber: clusterFailureBuildNumber, Severity: "hard", PR: failingPR, Job: failingTestLane},
				}},
				anotherClusterFailTestLane: &Details{Failed: 3, Jobs: []*Job{
					{BuildNumber: anotherClusterFailureBuildNumber, Severity: "hard", PR: failingPR, Job: anotherClusterFailTestLane},
				}},
			},
			"[rfe_id:1234][crit:high][owner:@sig-compute][test_id:4567]test case description": {
				failingTestLane: &Details{Failed: 4, Jobs: []*Job{
					{BuildNumber: clusterFailureBuildNumber, Severity: "hard", PR: failingPR, Job: failingTestLane},
				}},
				anotherClusterFailTestLane: &Details{Failed: 4, Jobs: []*Job{
					{BuildNumber: anotherClusterFailureBuildNumber, Severity: "hard", PR: failingPR, Job: anotherClusterFailTestLane},
				}},
			},
		}

		params := Params{
			Org:  "kubevirt",
			Repo: "kubevirt",
			Data: data,
			FailuresForJobs: map[int]*JobFailures{
				clusterFailureBuildNumber:        {
					BuildNumber: clusterFailureBuildNumber,
					PR:          failingPR,
					Job:         failingTestLane,
					Failures:    suspectedClusterFailureThreshold,
				},
				anotherClusterFailureBuildNumber: {
					BuildNumber: clusterFailureBuildNumber,
					PR:          failingPR,
					Job:         anotherClusterFailTestLane,
					Failures:    suspectedClusterFailureThreshold,
				},
			},
		}

		var ctrl *Controller
		var mockGithubClient *MockClient

		BeforeEach(func() {
			ctrl = NewController(GinkgoT())
			mockGithubClient = NewMockClient(ctrl)
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("stops after limit of creation has been reached", func() {
			mockGithubClient.EXPECT().FindIssues(Any(), Any(), Any()).Times(1)
			mockGithubClient.EXPECT().CreateIssue(Eq("kubevirt"), Eq("kubevirt"), Any(), Any(), Eq(0), Any(), Any()).Times(1)

			clusterFailureBuildNumbers, err := CreateClusterFailureIssues(params, suspectedClusterFailureThreshold, issueLabels, mockGithubClient, false, 1)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(clusterFailureBuildNumbers).To(gomega.HaveLen(2))
		})

		It("ignores limit if zero", func() {
			mockGithubClient.EXPECT().FindIssues(Any(), Any(), Any()).Times(2)
			mockGithubClient.EXPECT().CreateIssue(Eq("kubevirt"), Eq("kubevirt"), Any(), Any(), Eq(0), Any(), Any()).Times(2)

			clusterFailureBuildNumbers, err := CreateClusterFailureIssues(params, suspectedClusterFailureThreshold, issueLabels, mockGithubClient, false, 0)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(clusterFailureBuildNumbers).To(gomega.HaveLen(2))
		})

		It("uses org and repo when searching for and creating issues", func() {
			mockGithubClient.EXPECT().FindIssues(matchers.ContainsStrings("org:kubevirt", "repo:kubevirt"), Any(), Any()).Times(2)
			mockGithubClient.EXPECT().CreateIssue(Eq("kubevirt"), Eq("kubevirt"), Any(), Any(), Eq(0), Any(), Any()).Times(2)

			clusterFailureBuildNumbers, err := CreateClusterFailureIssues(params, suspectedClusterFailureThreshold, issueLabels, mockGithubClient, false, 0)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(clusterFailureBuildNumbers).To(gomega.HaveLen(2))
		})

		It("does not create cluster failures if failure threshold is zero", func() {
			mockGithubClient.EXPECT().FindIssues(Any(), Any(), Any()).Times(0)
			mockGithubClient.EXPECT().CreateIssue(Eq("kubevirt"), Eq("kubevirt"), Any(), Any(), Eq(0), Any(), Any()).Times(0)

			clusterFailureBuildNumbers, err := CreateClusterFailureIssues(params, 0, issueLabels, mockGithubClient, false, 0)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(clusterFailureBuildNumbers).To(gomega.BeNil())
		})

	})

})

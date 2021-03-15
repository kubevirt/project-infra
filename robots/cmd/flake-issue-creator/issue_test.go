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
	. "github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	gh "k8s.io/test-infra/prow/github"
	prowgithub "k8s.io/test-infra/prow/github"
	"strings"
	"time"

	. "kubevirt.io/project-infra/robots/cmd/flake-issue-creator"
	. "kubevirt.io/project-infra/robots/pkg/flakefinder"
	. "kubevirt.io/project-infra/robots/pkg/gomock/matchers"
	. "kubevirt.io/project-infra/robots/pkg/mock/prow/github"
)

var _ = Describe("issue.go", func() {

	When("extracting cluster failure issues", func() {

		It("returns err on labels not found", func() {
			labels, err := GetFlakeIssuesLabels(DefaultIssueLabels, []prowgithub.Label{}, "kubevirt", "kubevirt")
			gomega.Expect(err).ToNot(gomega.BeNil())
			gomega.Expect(labels).To(gomega.BeNil())
		})

		It("returns found labels", func() {
			labels := []prowgithub.Label{
				{Name: strings.Split(DefaultIssueLabels, ",")[0]},
				{Name: strings.Split(DefaultIssueLabels, ",")[1]},
			}

			issueLabels, err := GetFlakeIssuesLabels(DefaultIssueLabels, labels, "kubevirt", "kubevirt")
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(issueLabels).ToNot(gomega.BeNil())
			gomega.Expect(issueLabels).To(gomega.HaveLen(2))
		})

	})

	When("opening issues", func() {

		jobTestFailures := 10
		jobBuildNumber := 37
		clusterFailureJobBuildNumber := 666
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

		const existingIssueId = 42

		var ctrl *Controller
		var mockGithubClient *MockClient

		var issues []prowgithub.Issue

		skipExistingIssuesChangedLately := 12*time.Hour

		BeforeEach(func() {
			ctrl = NewController(GinkgoT())
			mockGithubClient = NewMockClient(ctrl)

			issues = NewFlakyTestIssues(params, []int{}, issueLabels)
		})

		AfterEach(func() {
			ctrl.Finish()
		})

		It("searches for issues within org and repo", func() {
			mockGithubClient.EXPECT().FindIssues(ContainsStrings("org:kubevirt", "repo:kubevirt"), Any(), Any()).Times(4)
			mockGithubClient.EXPECT().CreateIssue(Eq("kubevirt"), Eq("kubevirt"), Any(), Any(), Eq(0), Any(), Any()).Times(4)

			err := CreateIssues("kubevirt", "kubevirt", issueLabels, issues, mockGithubClient, false, skipExistingIssuesChangedLately)
			gomega.Expect(err).To(gomega.BeNil())
		})

		It("searches for issues with issue labels", func() {
			mockGithubClient.EXPECT().FindIssues(ContainsStrings("label:"+buildWatcher, "label:"+typeBug), Any(), Any()).Times(4)
			mockGithubClient.EXPECT().CreateIssue(Eq("kubevirt"), Eq("kubevirt"), Any(), Any(), Eq(0), Any(), Any()).Times(4)

			err := CreateIssues("kubevirt", "kubevirt", issueLabels, issues, mockGithubClient, false, skipExistingIssuesChangedLately)
			gomega.Expect(err).To(gomega.BeNil())
		})

		It("opens issues", func() {
			mockGithubClient.EXPECT().FindIssues(Any(), Any(), Any()).Times(4)
			mockGithubClient.EXPECT().CreateIssue(Eq("kubevirt"), Eq("kubevirt"), Any(), Any(), Eq(0), Any(), Any()).Times(4)

			err := CreateIssues("kubevirt", "kubevirt", issueLabels, issues, mockGithubClient, false, skipExistingIssuesChangedLately)
			gomega.Expect(err).To(gomega.BeNil())
		})

		It("does not open issues on dry run", func() {
			mockGithubClient.EXPECT().FindIssues(Any(), Any(), Any()).Times(4)
			mockGithubClient.EXPECT().CreateIssue(Eq("kubevirt"), Eq("kubevirt"), Any(), Any(), Eq(0), Any(), Any()).Times(0)

			err := CreateIssues("kubevirt", "kubevirt", issueLabels, issues, mockGithubClient, true, skipExistingIssuesChangedLately)
			gomega.Expect(err).To(gomega.BeNil())
		})

		It("adds comment when previous issue exists", func() {
			foundIssues := []gh.Issue{{ID: existingIssueId}}
			mockGithubClient.EXPECT().FindIssues(Any(), Any(), Any()).Return(foundIssues, nil).Times(1)
			mockGithubClient.EXPECT().FindIssues(Any(), Any(), Any()).Return(nil, nil).Times(3)
			mockGithubClient.EXPECT().CreateComment(Eq("kubevirt"), Eq("kubevirt"), existingIssueId, Any()).Times(1)
			mockGithubClient.EXPECT().CreateIssue(Eq("kubevirt"), Eq("kubevirt"), Any(), Any(), Eq(0), Any(), Any()).Times(3)

			err := CreateIssues("kubevirt", "kubevirt", issueLabels, issues, mockGithubClient, false, skipExistingIssuesChangedLately)
			gomega.Expect(err).To(gomega.BeNil())
		})

		It("reopens previous issue if exists", func() {
			foundIssues := []gh.Issue{{ID: existingIssueId, State: "closed"}}

			mockGithubClient.EXPECT().FindIssues(Any(), Any(), Any()).Return(foundIssues, nil).Times(1)
			mockGithubClient.EXPECT().FindIssues(Any(), Any(), Any()).Return(nil, nil).Times(3)
			mockGithubClient.EXPECT().ReopenIssue(Eq("kubevirt"), Eq("kubevirt"), existingIssueId).Times(1)
			mockGithubClient.EXPECT().CreateComment(Eq("kubevirt"), Eq("kubevirt"), existingIssueId, Any()).Times(1)
			mockGithubClient.EXPECT().CreateIssue(Eq("kubevirt"), Eq("kubevirt"), Any(), Any(), Eq(0), Any(), Any()).Times(3)

			err := CreateIssues("kubevirt", "kubevirt", issueLabels, issues, mockGithubClient, false, skipExistingIssuesChangedLately)
			gomega.Expect(err).To(gomega.BeNil())
		})

		It("does not modify previous issues on dry run", func() {
			foundIssues := []gh.Issue{{ID: existingIssueId, State: "closed"}}

			mockGithubClient.EXPECT().FindIssues(Any(), Any(), Any()).Return(foundIssues, nil).Times(4)
			mockGithubClient.EXPECT().CreateComment(Eq("kubevirt"), Eq("kubevirt"), existingIssueId, Any()).Times(0)
			mockGithubClient.EXPECT().ReopenIssue(Eq("kubevirt"), Eq("kubevirt"), existingIssueId).Times(0)
			mockGithubClient.EXPECT().CreateIssue(Eq("kubevirt"), Eq("kubevirt"), Any(), Any(), Eq(0), Any(), Any()).Times(0)

			err := CreateIssues("kubevirt", "kubevirt", issueLabels, issues, mockGithubClient, true, skipExistingIssuesChangedLately)
			gomega.Expect(err).To(gomega.BeNil())
		})

		It("does not comment on issues that were created less than x hours ago", func() {
			foundIssues := []gh.Issue{{ID: existingIssueId, CreatedAt: time.Now().Add(-1*time.Hour*6)}}
			mockGithubClient.EXPECT().FindIssues(Any(), Any(), Any()).Return(foundIssues, nil).Times(1)
			mockGithubClient.EXPECT().FindIssues(Any(), Any(), Any()).Return(nil, nil).Times(3)
			mockGithubClient.EXPECT().CreateComment(Eq("kubevirt"), Eq("kubevirt"), existingIssueId, Any()).Times(0)
			mockGithubClient.EXPECT().CreateIssue(Eq("kubevirt"), Eq("kubevirt"), Any(), Any(), Eq(0), Any(), Any()).Times(3)

			err := CreateIssues("kubevirt", "kubevirt", issueLabels, issues, mockGithubClient, false, skipExistingIssuesChangedLately)
			gomega.Expect(err).To(gomega.BeNil())
		})

		It("does not comment on issues that were updated less than x hours ago", func() {
			foundIssues := []gh.Issue{{ID: existingIssueId, UpdatedAt: time.Now().Add(-1*time.Hour*6)}}
			mockGithubClient.EXPECT().FindIssues(Any(), Any(), Any()).Return(foundIssues, nil).Times(1)
			mockGithubClient.EXPECT().FindIssues(Any(), Any(), Any()).Return(nil, nil).Times(3)
			mockGithubClient.EXPECT().CreateComment(Eq("kubevirt"), Eq("kubevirt"), existingIssueId, Any()).Times(0)
			mockGithubClient.EXPECT().CreateIssue(Eq("kubevirt"), Eq("kubevirt"), Any(), Any(), Eq(0), Any(), Any()).Times(3)

			err := CreateIssues("kubevirt", "kubevirt", issueLabels, issues, mockGithubClient, false, skipExistingIssuesChangedLately)
			gomega.Expect(err).To(gomega.BeNil())
		})

	})

	searchLabels := []prowgithub.Label{
		{Name: "whatever"},
	}

	When("creating the query string", func() {

		It("uses org, repo, labels and title", func() {
			query, err := CreateFindIssuesQuery("myorg", "myrepo", prowgithub.Issue{Title: "issue title"}, searchLabels)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(query, err).To(gomega.BeEquivalentTo(
				"org:myorg repo:myrepo label:whatever \"issue title\"",
			))
		})

		It("cuts the query string down to 256 characters", func() {
			query, err := CreateFindIssuesQuery("myorg", "myrepo", prowgithub.Issue{Title: "issue title issue title issue title issue title issue title issue title issue title issue title issue title issue title issue title issue title issue title issue title issue title issue title issue title issue title issue title issue title issue title issue title issue title issue title "}, searchLabels)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(len(query) <= 256).To(gomega.BeTrue())
		})

		It("can handle to cut down even with no spaces", func() {
			query, err := CreateFindIssuesQuery("myorg", "myrepo", prowgithub.Issue{Title: "[issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title]"}, searchLabels)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(len(query) <= 256).To(gomega.BeTrue())
		})

		It("cant handle strings with word characters only", func() {
			query, err := CreateFindIssuesQuery("myorg", "myrepo", prowgithub.Issue{Title: "issuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitleissuetitle"}, searchLabels)
			gomega.Expect(err).ToNot(gomega.BeNil())
			gomega.Expect(query).To(gomega.BeEquivalentTo(""))
		})

		It("test_id takes precedence in query", func() {
			query, err := CreateFindIssuesQuery("myorg", "myrepo", prowgithub.Issue{Title: "[issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][issue][title][test_id:1742]"}, searchLabels)
			fmt.Println(query)
			gomega.Expect(err).To(gomega.BeNil())
			gomega.Expect(len(query) <= 256).To(gomega.BeTrue())
			gomega.Expect(query).To(gomega.ContainSubstring("\"test_id:1742\""))
		})

	})

})

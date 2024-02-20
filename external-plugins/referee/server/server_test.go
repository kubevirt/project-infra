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

package server

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"k8s.io/test-infra/prow/github"
	"kubevirt.io/project-infra/external-plugins/referee/ghgraphql"
)

const (
	org      = "org"
	repo     = "repo"
	prNumber = 1742
	user     = "user"
	botuser  = "botuser"
)

type fakeGitHubClient struct {
	mock.Mock
}

func newFakeGitHubClient() *fakeGitHubClient {
	return &fakeGitHubClient{}
}

func (_m *fakeGitHubClient) CreateComment(org, repo string, number int, comment string) error {
	arguments := _m.Called(org, repo, number, comment)
	return arguments.Error(0)
}

type fakeGitHubGraphQLClient struct {
	mock.Mock
}

func newFakeGitHubGraphQLClient() *fakeGitHubGraphQLClient {
	return &fakeGitHubGraphQLClient{}
}

func (_m *fakeGitHubGraphQLClient) FetchPRTimeLineForLastCommit(org string, repo string, prNumber int) (ghgraphql.PRTimelineForLastCommit, error) {
	arguments := _m.Called(org, repo, prNumber)
	return arguments.Get(0).(ghgraphql.PRTimelineForLastCommit), arguments.Error(1)
}

var _ = Describe("", func() {
	Context("handlePullRequestComment", func() {
		var server Server
		var mockGitHubClient *fakeGitHubClient
		var mockGitHubGraphQLClient *fakeGitHubGraphQLClient
		BeforeEach(func() {
			entry := logrus.StandardLogger().WithFields(map[string]interface{}{"type": "testlogger"})
			mockGitHubClient = newFakeGitHubClient()
			mockGitHubGraphQLClient = newFakeGitHubGraphQLClient()
			server = Server{
				TokenGenerator:  nil,
				BotName:         botuser,
				Log:             entry,
				GithubClient:    mockGitHubClient,
				GHGraphQLClient: mockGitHubGraphQLClient,
				DryRun:          false,
			}
		})
		It("doesn't handle other action", func() {
			Expect(server.handlePullRequestComment(github.IssueCommentEvent{
				Action: github.IssueCommentActionDeleted,
				Issue:  github.Issue{},
				Comment: github.IssueComment{
					User: github.User{Login: user},
				},
				Repo: github.Repo{},
				GUID: "",
			})).ShouldNot(HaveOccurred())
			mockGitHubGraphQLClient.AssertExpectations(GinkgoT())
			mockGitHubClient.AssertExpectations(GinkgoT())
		})
		It("doesn't handle issue comment", func() {
			Expect(server.handlePullRequestComment(github.IssueCommentEvent{
				Action: github.IssueCommentActionCreated,
				Issue:  github.Issue{},
				Comment: github.IssueComment{
					User: github.User{Login: user},
					Body: `This is an issue comment

/help-wanted
`,
				},
				Repo: github.Repo{},
				GUID: "",
			})).ShouldNot(HaveOccurred())
			mockGitHubGraphQLClient.AssertExpectations(GinkgoT())
			mockGitHubClient.AssertExpectations(GinkgoT())
		})
		It("doesn't handle comment if it doesn't contain a test command", func() {
			Expect(server.handlePullRequestComment(github.IssueCommentEvent{
				Action: github.IssueCommentActionCreated,
				Issue: github.Issue{
					Number:      prNumber,
					PullRequest: &struct{}{},
				},
				Comment: github.IssueComment{
					User: github.User{Login: user},
					Body: `This is a comment on a pull request but not a test command

/help-wanted
`,
				},
				Repo: github.Repo{
					Owner: github.User{Login: org},
					Name:  repo,
				},
				GUID: "",
			})).ShouldNot(HaveOccurred())
			mockGitHubGraphQLClient.AssertExpectations(GinkgoT())
			mockGitHubClient.AssertExpectations(GinkgoT())
		})
		It("fetches number of retests on test all comment, but doesn't place comment since not enough retest comments found", func() {
			mockGitHubGraphQLClient.On(
				"FetchPRTimeLineForLastCommit", org, repo, prNumber,
			).Return(
				ghgraphql.PRTimelineForLastCommit{NumberOfRetestComments: 4}, nil,
			)
			Expect(server.handlePullRequestComment(github.IssueCommentEvent{
				Action: github.IssueCommentActionCreated,
				Issue: github.Issue{
					Number:      prNumber,
					PullRequest: &struct{}{},
				},
				Comment: github.IssueComment{
					User: github.User{Login: user},
					Body: `This is a comment triggering a test on a pull request

/test all
`,
				},
				Repo: github.Repo{
					Owner: github.User{Login: org},
					Name:  repo,
				},
				GUID: "",
			})).ShouldNot(HaveOccurred())
			mockGitHubGraphQLClient.AssertExpectations(GinkgoT())
			mockGitHubClient.AssertExpectations(GinkgoT())
		})
		It("fetches number of retests on test-all comment, then places comment", func() {
			mockGitHubGraphQLClient.On(
				"FetchPRTimeLineForLastCommit", org, repo, prNumber).Return(ghgraphql.PRTimelineForLastCommit{NumberOfRetestComments: 5}, nil)
			mockGitHubClient.On("CreateComment", org, repo, prNumber, mock.Anything).Return(nil)
			Expect(server.handlePullRequestComment(github.IssueCommentEvent{
				Action: github.IssueCommentActionCreated,
				Issue: github.Issue{
					Number:      prNumber,
					PullRequest: &struct{}{},
				},
				Comment: github.IssueComment{
					User: github.User{Login: user},
					Body: `This is a comment triggering a test on a pull request

/test all
`,
				},
				Repo: github.Repo{
					Owner: github.User{Login: org},
					Name:  repo,
				},
				GUID: "",
			})).ShouldNot(HaveOccurred())
			mockGitHubGraphQLClient.AssertExpectations(GinkgoT())
			mockGitHubClient.AssertExpectations(GinkgoT())
		})
		It("fetches number of retests on retest-required comment, then places comment", func() {
			mockGitHubGraphQLClient.On(
				"FetchPRTimeLineForLastCommit", org, repo, prNumber).Return(ghgraphql.PRTimelineForLastCommit{NumberOfRetestComments: 5}, nil)
			mockGitHubClient.On("CreateComment", org, repo, prNumber, mock.Anything).Return(nil)
			Expect(server.handlePullRequestComment(github.IssueCommentEvent{
				Action: github.IssueCommentActionCreated,
				Issue: github.Issue{
					Number:      prNumber,
					PullRequest: &struct{}{},
				},
				Comment: github.IssueComment{
					User: github.User{Login: user},
					Body: `This is a comment triggering a test on a pull request

/retest-required
`,
				},
				Repo: github.Repo{
					Owner: github.User{Login: org},
					Name:  repo,
				},
				GUID: "",
			})).ShouldNot(HaveOccurred())
			mockGitHubGraphQLClient.AssertExpectations(GinkgoT())
			mockGitHubClient.AssertExpectations(GinkgoT())
		})
		It("fetches number of retests on retest-required comment, but doesn't place comment if hold present", func() {
			mockGitHubGraphQLClient.On(
				"FetchPRTimeLineForLastCommit", org, repo, prNumber,
			).Return(
				ghgraphql.PRTimelineForLastCommit{NumberOfRetestComments: 5, WasHeld: true}, nil,
			)
			Expect(server.handlePullRequestComment(github.IssueCommentEvent{
				Action: github.IssueCommentActionCreated,
				Issue: github.Issue{
					Number:      prNumber,
					PullRequest: &struct{}{},
				},
				Comment: github.IssueComment{
					User: github.User{Login: user},
					Body: `This is a comment triggering a test on a pull request

/retest-required
`,
				},
				Repo: github.Repo{
					Owner: github.User{Login: org},
					Name:  repo,
				},
				GUID: "",
			})).ShouldNot(HaveOccurred())
			mockGitHubGraphQLClient.AssertExpectations(GinkgoT())
			mockGitHubClient.AssertExpectations(GinkgoT())
		})
		It("handles test-all comment if the bot user is the commenter", func() {
			mockGitHubGraphQLClient.On(
				"FetchPRTimeLineForLastCommit", org, repo, prNumber).Return(ghgraphql.PRTimelineForLastCommit{NumberOfRetestComments: 5}, nil)
			mockGitHubClient.On("CreateComment", org, repo, prNumber, mock.Anything).Return(nil)
			Expect(server.handlePullRequestComment(github.IssueCommentEvent{
				Action: github.IssueCommentActionCreated,
				Issue: github.Issue{
					Number:      prNumber,
					PullRequest: &struct{}{},
				},
				Comment: github.IssueComment{
					User: github.User{Login: botuser},
					Body: `This is a comment triggering a test on a pull request

/test all
`,
				},
				Repo: github.Repo{
					Owner: github.User{Login: org},
					Name:  repo,
				},
				GUID: "",
			})).ShouldNot(HaveOccurred())
			mockGitHubGraphQLClient.AssertExpectations(GinkgoT())
			mockGitHubClient.AssertExpectations(GinkgoT())
		})
		It("doesn't handle comment if the bot user is the commenter but there's no test trigger", func() {
			Expect(server.handlePullRequestComment(github.IssueCommentEvent{
				Action: github.IssueCommentActionCreated,
				Issue: github.Issue{
					Number:      prNumber,
					PullRequest: &struct{}{},
				},
				Comment: github.IssueComment{
					User: github.User{Login: botuser},
					Body: `This is a comment asking for help on the test plugin

/test ?
`,
				},
				Repo: github.Repo{
					Owner: github.User{Login: org},
					Name:  repo,
				},
				GUID: "",
			})).ShouldNot(HaveOccurred())
			mockGitHubGraphQLClient.AssertExpectations(GinkgoT())
			mockGitHubClient.AssertExpectations(GinkgoT())
		})
	})
})

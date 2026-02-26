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
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/github"
)

const (
	testOrg  = "kubevirt"
	testRepo = "kubevirt"
)

type fakeGitHubClient struct {
	labels          map[string][]github.Label
	commits         map[string][]github.RepositoryCommit
	labelsAdded     []string
	labelsRemoved   []string
}

func newFakeGitHubClient() *fakeGitHubClient {
	return &fakeGitHubClient{
		labels:  make(map[string][]github.Label),
		commits: make(map[string][]github.RepositoryCommit),
	}
}

func prKey(org, repo string, number int) string {
	return fmt.Sprintf("%s/%s#%d", org, repo, number)
}

func (f *fakeGitHubClient) AddLabel(org, repo string, number int, label string) error {
	f.labelsAdded = append(f.labelsAdded, label)
	key := prKey(org, repo, number)
	f.labels[key] = append(f.labels[key], github.Label{Name: label})
	return nil
}

func (f *fakeGitHubClient) RemoveLabel(org, repo string, number int, label string) error {
	f.labelsRemoved = append(f.labelsRemoved, label)
	key := prKey(org, repo, number)
	var filtered []github.Label
	for _, l := range f.labels[key] {
		if l.Name != label {
			filtered = append(filtered, l)
		}
	}
	f.labels[key] = filtered
	return nil
}

func (f *fakeGitHubClient) GetIssueLabels(org, repo string, number int) ([]github.Label, error) {
	return f.labels[prKey(org, repo, number)], nil
}

func (f *fakeGitHubClient) ListPullRequestCommits(org, repo string, number int) ([]github.RepositoryCommit, error) {
	return f.commits[prKey(org, repo, number)], nil
}

func makeCommit(message string) github.RepositoryCommit {
	return github.RepositoryCommit{
		Commit: github.GitCommit{
			Message: message,
		},
	}
}

func makePREvent(org, repo string, number int, action github.PullRequestEventAction) github.PullRequestEvent {
	return github.PullRequestEvent{
		Action: action,
		Number: number,
		Repo: github.Repo{
			Owner: github.User{Login: org},
			Name:  repo,
		},
	}
}

var _ = Describe("ai-label", func() {
	var s Server
	var ghc *fakeGitHubClient
	var log *logrus.Entry

	BeforeEach(func() {
		log = logrus.StandardLogger().WithField("test", true)
		ghc = newFakeGitHubClient()
		s = Server{
			Log:          log,
			GithubClient: ghc,
			DryRun:       false,
			Patterns:     DefaultPatterns(),
		}
	})

	Context("matchCommits", func() {
		It("matches Co-Authored-By Claude trailer", func() {
			commits := []github.RepositoryCommit{
				makeCommit("Fix bug\n\nCo-Authored-By: Claude <noreply@anthropic.com>"),
			}
			matched := s.matchCommits(commits)
			Expect(matched).To(HaveKey("ai/claude"))
			Expect(matched).To(HaveLen(1))
		})

		It("matches Assisted-by Claude trailer", func() {
			commits := []github.RepositoryCommit{
				makeCommit("Fix bug\n\nAssisted-by: Claude <noreply@anthropic.com>"),
			}
			matched := s.matchCommits(commits)
			Expect(matched).To(HaveKey("ai/claude"))
		})

		It("matches Generated-by Claude trailer", func() {
			commits := []github.RepositoryCommit{
				makeCommit("Fix bug\n\nGenerated-by: Claude <noreply@anthropic.com>"),
			}
			matched := s.matchCommits(commits)
			Expect(matched).To(HaveKey("ai/claude"))
		})

		It("matches Claude with version info in trailer", func() {
			commits := []github.RepositoryCommit{
				makeCommit("Fix bug\n\nCo-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"),
			}
			matched := s.matchCommits(commits)
			Expect(matched).To(HaveKey("ai/claude"))
		})

		It("matches Cursor trailer", func() {
			commits := []github.RepositoryCommit{
				makeCommit("Fix bug\n\nCo-Authored-By: Cursor <cursor@cursor.com>"),
			}
			matched := s.matchCommits(commits)
			Expect(matched).To(HaveKey("ai/cursor"))
			Expect(matched).To(HaveLen(1))
		})

		It("matches Copilot trailer", func() {
			commits := []github.RepositoryCommit{
				makeCommit("Fix bug\n\nCo-Authored-By: Copilot <copilot@github.com>"),
			}
			matched := s.matchCommits(commits)
			Expect(matched).To(HaveKey("ai/copilot"))
			Expect(matched).To(HaveLen(1))
		})

		It("matches multiple AI tools across commits", func() {
			commits := []github.RepositoryCommit{
				makeCommit("Fix bug\n\nCo-Authored-By: Claude <noreply@anthropic.com>"),
				makeCommit("Add feature\n\nCo-Authored-By: Copilot <copilot@github.com>"),
			}
			matched := s.matchCommits(commits)
			Expect(matched).To(HaveKey("ai/claude"))
			Expect(matched).To(HaveKey("ai/copilot"))
			Expect(matched).To(HaveLen(2))
		})

		It("deduplicates labels across commits", func() {
			commits := []github.RepositoryCommit{
				makeCommit("Fix bug\n\nCo-Authored-By: Claude <noreply@anthropic.com>"),
				makeCommit("Fix another bug\n\nAssisted-by: Claude <noreply@anthropic.com>"),
			}
			matched := s.matchCommits(commits)
			Expect(matched).To(HaveKey("ai/claude"))
			Expect(matched).To(HaveLen(1))
		})

		It("returns empty map for commits without AI trailers", func() {
			commits := []github.RepositoryCommit{
				makeCommit("Fix bug\n\nSigned-off-by: Developer <dev@example.com>"),
			}
			matched := s.matchCommits(commits)
			Expect(matched).To(BeEmpty())
		})

		It("is case-insensitive for trailer names", func() {
			commits := []github.RepositoryCommit{
				makeCommit("Fix bug\n\nco-authored-by: Claude <noreply@anthropic.com>"),
			}
			matched := s.matchCommits(commits)
			Expect(matched).To(HaveKey("ai/claude"))
		})

		It("handles empty commit list", func() {
			matched := s.matchCommits(nil)
			Expect(matched).To(BeEmpty())
		})
	})

	Context("handlePullRequest", func() {
		It("adds labels for matching commits", func() {
			key := prKey(testOrg, testRepo, 1)
			ghc.commits[key] = []github.RepositoryCommit{
				makeCommit("Fix bug\n\nCo-Authored-By: Claude <noreply@anthropic.com>"),
			}

			pr := makePREvent(testOrg, testRepo, 1, github.PullRequestActionOpened)
			err := s.handlePullRequest(log, pr)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ghc.labelsAdded).To(ContainElement("ai/claude"))
		})

		It("does not add labels that already exist", func() {
			key := prKey(testOrg, testRepo, 1)
			ghc.commits[key] = []github.RepositoryCommit{
				makeCommit("Fix bug\n\nCo-Authored-By: Claude <noreply@anthropic.com>"),
			}
			ghc.labels[key] = []github.Label{{Name: "ai/claude"}}

			pr := makePREvent(testOrg, testRepo, 1, github.PullRequestActionOpened)
			err := s.handlePullRequest(log, pr)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ghc.labelsAdded).To(BeEmpty())
		})

		It("removes stale ai/* labels after force-push", func() {
			key := prKey(testOrg, testRepo, 1)
			ghc.commits[key] = []github.RepositoryCommit{
				makeCommit("Fix bug\n\nSigned-off-by: Developer <dev@example.com>"),
			}
			ghc.labels[key] = []github.Label{{Name: "ai/claude"}}

			pr := makePREvent(testOrg, testRepo, 1, github.PullRequestActionSynchronize)
			err := s.handlePullRequest(log, pr)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ghc.labelsRemoved).To(ContainElement("ai/claude"))
		})

		It("skips non-relevant PR actions", func() {
			pr := makePREvent(testOrg, testRepo, 1, github.PullRequestActionClosed)
			err := s.handlePullRequest(log, pr)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ghc.labelsAdded).To(BeEmpty())
			Expect(ghc.labelsRemoved).To(BeEmpty())
		})

		It("handles reopened action", func() {
			key := prKey(testOrg, testRepo, 1)
			ghc.commits[key] = []github.RepositoryCommit{
				makeCommit("Fix bug\n\nCo-Authored-By: Claude <noreply@anthropic.com>"),
			}

			pr := makePREvent(testOrg, testRepo, 1, github.PullRequestActionReopened)
			err := s.handlePullRequest(log, pr)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ghc.labelsAdded).To(ContainElement("ai/claude"))
		})

		It("does not modify labels in dry-run mode", func() {
			s.DryRun = true
			key := prKey(testOrg, testRepo, 1)
			ghc.commits[key] = []github.RepositoryCommit{
				makeCommit("Fix bug\n\nCo-Authored-By: Claude <noreply@anthropic.com>"),
			}

			pr := makePREvent(testOrg, testRepo, 1, github.PullRequestActionOpened)
			err := s.handlePullRequest(log, pr)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ghc.labelsAdded).To(BeEmpty())
		})

		It("does not touch non-ai labels", func() {
			key := prKey(testOrg, testRepo, 1)
			ghc.commits[key] = []github.RepositoryCommit{
				makeCommit("Fix bug"),
			}
			ghc.labels[key] = []github.Label{
				{Name: "kind/bug"},
				{Name: "lgtm"},
			}

			pr := makePREvent(testOrg, testRepo, 1, github.PullRequestActionSynchronize)
			err := s.handlePullRequest(log, pr)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ghc.labelsRemoved).To(BeEmpty())
		})

		It("adds and removes labels in the same event", func() {
			key := prKey(testOrg, testRepo, 1)
			ghc.commits[key] = []github.RepositoryCommit{
				makeCommit("Fix bug\n\nCo-Authored-By: Copilot <copilot@github.com>"),
			}
			ghc.labels[key] = []github.Label{{Name: "ai/claude"}}

			pr := makePREvent(testOrg, testRepo, 1, github.PullRequestActionSynchronize)
			err := s.handlePullRequest(log, pr)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(ghc.labelsAdded).To(ContainElement("ai/copilot"))
			Expect(ghc.labelsRemoved).To(ContainElement("ai/claude"))
		})
	})

	Context("handleEvent", func() {
		It("ignores non-pull_request events", func() {
			payload := []byte(`{}`)
			err := s.handleEvent("issue_comment", "guid-123", payload)
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})

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
 */

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/pluginhelp"
)

const PluginName = "ai-label"

// AIPattern maps a compiled regex to the label that should be applied when matched.
type AIPattern struct {
	Regex *regexp.Regexp
	Label string
}

// DefaultPatterns returns the default set of AI attribution patterns.
func DefaultPatterns() []AIPattern {
	return []AIPattern{
		{
			Regex: regexp.MustCompile(`(?im)^(Co-Authored-By|Assisted-by|Generated-by):\s*Claude.*<.*noreply@anthropic\.com.*>`),
			Label: "ai/claude",
		},
		{
			Regex: regexp.MustCompile(`(?im)^Co-Authored-By:\s*.*Cursor.*<.*cursor.*>`),
			Label: "ai/cursor",
		},
		{
			Regex: regexp.MustCompile(`(?im)^Co-Authored-By:\s*.*Copilot.*<.*github\.com.*>`),
			Label: "ai/copilot",
		},
	}
}

type githubClient interface {
	AddLabel(org, repo string, number int, label string) error
	RemoveLabel(org, repo string, number int, label string) error
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	ListPullRequestCommits(org, repo string, number int) ([]github.RepositoryCommit, error)
}

// HelpProvider constructs the pluginhelp.PluginHelp for this plugin.
func HelpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The ai-label plugin automatically applies ai/* labels to PRs that contain commits with AI attribution trailers (e.g. Co-Authored-By: Claude <noreply@anthropic.com>).`,
	}
	return pluginHelp, nil
}

// Server implements http.Handler. It validates incoming GitHub webhooks and
// then dispatches them to the appropriate handler.
type Server struct {
	TokenGenerator func() []byte
	Log            *logrus.Entry
	GithubClient   githubClient
	DryRun         bool
	Patterns       []AIPattern
}

// ServeHTTP validates an incoming webhook and puts it into the event channel.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.TokenGenerator)
	if !ok {
		return
	}
	fmt.Fprint(w, "Event received. Have a nice day.")

	if err := s.handleEvent(eventType, eventGUID, payload); err != nil {
		s.Log.WithError(err).Error("Error parsing event.")
	}
}

func (s *Server) handleEvent(eventType, eventGUID string, payload []byte) error {
	l := s.Log.WithFields(logrus.Fields{
		"event-type":     eventType,
		github.EventGUID: eventGUID,
	})
	switch eventType {
	case "pull_request":
		var pr github.PullRequestEvent
		if err := json.Unmarshal(payload, &pr); err != nil {
			return err
		}
		go func(pr github.PullRequestEvent) {
			if err := s.handlePullRequest(l, pr); err != nil {
				s.Log.WithError(err).WithFields(l.Data).Errorf("%s failed.", PluginName)
			}
		}(pr)
	default:
		l.Debugf("skipping event of type %q", eventType)
	}
	return nil
}

func (s *Server) handlePullRequest(l *logrus.Entry, pr github.PullRequestEvent) error {
	org := pr.Repo.Owner.Login
	repo := pr.Repo.Name
	num := pr.Number
	action := pr.Action

	switch action {
	case github.PullRequestActionOpened, github.PullRequestActionSynchronize, github.PullRequestActionReopened:
	default:
		l.Debugf("skipping pull_request action %s", action)
		return nil
	}

	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  org,
		github.RepoLogField: repo,
		github.PrLogField:   num,
	})

	commits, err := s.GithubClient.ListPullRequestCommits(org, repo, num)
	if err != nil {
		return fmt.Errorf("failed to list PR commits for %s/%s#%d: %w", org, repo, num, err)
	}

	matched := s.matchCommits(commits)

	currentLabels, err := s.GithubClient.GetIssueLabels(org, repo, num)
	if err != nil {
		return fmt.Errorf("failed to get labels for %s/%s#%d: %w", org, repo, num, err)
	}

	currentAILabels := make(map[string]bool)
	for _, label := range currentLabels {
		if strings.HasPrefix(label.Name, "ai/") {
			currentAILabels[label.Name] = true
		}
	}

	// Add missing labels
	for label := range matched {
		if !currentAILabels[label] {
			l.Infof("adding label %s", label)
			if !s.DryRun {
				if err := s.GithubClient.AddLabel(org, repo, num, label); err != nil {
					return fmt.Errorf("failed to add label %s to %s/%s#%d: %w", label, org, repo, num, err)
				}
			}
		}
	}

	// Remove stale ai/* labels
	for label := range currentAILabels {
		if !matched[label] {
			l.Infof("removing stale label %s", label)
			if !s.DryRun {
				if err := s.GithubClient.RemoveLabel(org, repo, num, label); err != nil {
					return fmt.Errorf("failed to remove label %s from %s/%s#%d: %w", label, org, repo, num, err)
				}
			}
		}
	}

	return nil
}

// matchCommits scans all commit messages against configured patterns and returns
// the set of labels that should be applied.
func (s *Server) matchCommits(commits []github.RepositoryCommit) map[string]bool {
	matched := make(map[string]bool)
	for _, commit := range commits {
		msg := commit.Commit.Message
		for _, pattern := range s.Patterns {
			if pattern.Regex.MatchString(msg) {
				matched[pattern.Label] = true
			}
		}
	}
	return matched
}

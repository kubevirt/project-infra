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
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"kubevirt.io/project-infra/external-plugins/referee/ghgraphql"
	"net/http"
)

const PluginName = "referee"

//go:embed tooManyRetestsComment.md
var tooManyRetestsComment string

// HelpProvider construct the pluginhelp.PluginHelp for this plugin.
func HelpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The referee plugin ensures users follow the rules of ci. It looks for rule violations and gives out warnings and penalties.`,
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/referee",
		Description: "Trigger referee review of a PR.",
		Featured:    true,
		WhoCanUse:   "Anyone",
		Examples:    []string{"/referee"},
	})
	return pluginHelp, nil
}

// Server implements http.Handler. It validates incoming GitHub webhooks and
// then dispatches them to the appropriate plugins.
type Server struct {
	TokenGenerator func() []byte
	BotName        string

	Log *logrus.Entry

	GithubClient    github.Client
	GHGraphQLClient *githubv4.Client

	// Whether to create comments on PRs or to just write them to the log
	DryRun bool
}

// ServeHTTP validates an incoming webhook and puts it into the event channel.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.TokenGenerator)
	if !ok {
		return
	}

	if err := s.handleEvent(eventType, eventGUID, payload); err != nil {
		s.Log.WithError(err).Error("Error parsing event.")
	}
}

func (s *Server) handleEvent(eventType, eventGUID string, payload []byte) error {
	l := logrus.WithFields(
		logrus.Fields{
			"event-type":     eventType,
			github.EventGUID: eventGUID,
		},
	)
	switch eventType {
	//https://docs.github.com/de/webhooks/webhook-events-and-payloads#issue_comment
	case "issue_comment":
		var ic github.IssueCommentEvent
		if err := json.Unmarshal(payload, &ic); err != nil {
			return err
		}
		go func() {
			if err := s.handlePullRequestComment(ic); err != nil {
				s.Log.WithError(err).WithFields(l.Data).Infof("%s failed.", PluginName)
			}
		}()
	default:
		s.Log.WithFields(l.Data).Debugf("skipping event of type %q", eventType)
	}
	return nil
}

func (s *Server) handlePullRequestComment(ic github.IssueCommentEvent) error {
	org := ic.Repo.Owner.Login
	repo := ic.Repo.Name
	num := ic.Issue.Number

	if !ic.Issue.IsPullRequest() {
		s.Log.Debugf("skipping referee since %s/%s#%d is not a pull request", org, repo, num)
		return nil
	}

	user := ic.Comment.User.Login
	action := ic.Action

	switch action {
	case github.IssueCommentActionCreated:
	default:
		s.Log.Debugf("skipping referee for action %s on pull request %s/%s#%d by %s", action, org, repo, num, user)
		return nil
	}
	numberOfRetestCommentsForLatestCommit, err := ghgraphql.FetchNumberOfRetestCommentsForLatestCommit(s.GHGraphQLClient, org, repo, num)
	if err != nil {
		s.Log.Fatalf("failed to fetch number of retest comments for pr %s/%s#%d: %v", org, repo, num, err)
	}

	if numberOfRetestCommentsForLatestCommit < 5 {
		s.Log.Debugf("number of retest comments for pr %s/%s#%d: %d", org, repo, num, numberOfRetestCommentsForLatestCommit)
		return nil
	}

	if !s.DryRun {
		err = s.GithubClient.CreateComment(org, repo, num, tooManyRetestsComment)
		if err != nil {
			return fmt.Errorf("error while creating review comment: %v", err)
		}
	} else {
		s.Log.Warnf("excessive number of retest comments for pr %s/%s#%d: %d", org, repo, num, numberOfRetestCommentsForLatestCommit)
	}
	return nil
}

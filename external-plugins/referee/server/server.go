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
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"html/template"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pjutil"
	"k8s.io/test-infra/prow/pluginhelp"
	"kubevirt.io/project-infra/external-plugins/referee/ghgraphql"
	"net/http"
)

const PluginName = "referee"

type TooManyRequestsData struct {
	Author string
	Team   string
}

//go:embed tooManyRetestsComment.gomd
var tooManyRetestsCommentTemplateBase string
var tooManyRetestsCommentTemplate *template.Template

func init() {
	var err error
	tooManyRetestsCommentTemplate, err = template.New("TooManyRequestsCommentTemplate").Parse(tooManyRetestsCommentTemplateBase)
	if err != nil {
		panic(fmt.Errorf("couldn't parse tooManyRetestsComment.gomd: %w", err))
	}
}

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

type githubClient interface {
	CreateComment(org, repo string, number int, comment string) error
}

// Server implements http.Handler. It validates incoming GitHub webhooks and
// then dispatches them to the appropriate plugins.
type Server struct {
	TokenGenerator func() []byte
	BotName        string

	Log *logrus.Entry

	GithubClient    githubClient
	GHGraphQLClient ghgraphql.GitHubGraphQLClient

	// DryRun says whether to create comments on PRs or to just write them to the log
	DryRun bool

	// Team is the name of the GitHub team that should be pinged
	Team string
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
	//https://docs.github.com/webhooks/webhook-events-and-payloads#issue_comment
	case "issue_comment":
		var ic github.IssueCommentEvent
		if err := json.Unmarshal(payload, &ic); err != nil {
			return err
		}
		go func() {
			if err := s.handlePullRequestComment(ic); err != nil {
				s.Log.WithError(err).WithFields(l.Data).Errorf("%s failed.", PluginName)
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
	action := ic.Action
	user := ic.Comment.User.Login

	switch action {
	case github.IssueCommentActionCreated:
	default:
		s.Log.Debugf("skipping referee for action %s on pull request %s/%s#%d by %s", action, org, repo, num, user)
		return nil
	}

	if !ic.Issue.IsPullRequest() {
		s.Log.Debugf("skipping referee since %s/%s#%d is not a pull request", org, repo, num)
		return nil
	}

	if !pjutil.RetestRe.MatchString(ic.Comment.Body) &&
		!pjutil.RetestRequiredRe.MatchString(ic.Comment.Body) &&
		!pjutil.TestAllRe.MatchString(ic.Comment.Body) {
		s.Log.Debugf("skipping referee since %s/%s#%d comment didn't contain command triggering tests", org, repo, num)
		return nil
	}

	prTimeLineForLastCommit, err := s.GHGraphQLClient.FetchPRTimeLineForLastCommit(org, repo, num)
	if err != nil {
		s.Log.Fatalf("failed to fetch number of retest comments for pr %s/%s#%d: %v", org, repo, num, err)
	}

	if prTimeLineForLastCommit.NumberOfRetestComments < 5 {
		s.Log.Debugf("skipping referee due to less number of retest comments for pr %s/%s#%d: %v", org, repo, num, prTimeLineForLastCommit)
		return nil
	}

	if prTimeLineForLastCommit.WasHeld {
		s.Log.Debugf("skipping referee due to hold present for pr %s/%s#%d: %v", org, repo, num, prTimeLineForLastCommit)
		return nil
	}

	if !s.DryRun {
		var output bytes.Buffer
		err := tooManyRetestsCommentTemplate.Execute(&output, TooManyRequestsData{
			Author: user,
			Team:   s.Team,
		})
		if err != nil {
			return fmt.Errorf("error while rendering comment template: %v", err)
		}
		err = s.GithubClient.CreateComment(org, repo, num, output.String())
		if err != nil {
			return fmt.Errorf("error while creating review comment: %v", err)
		}
	} else {
		s.Log.Warnf("excessive number of retest comments for pr %s/%s#%d: %v", org, repo, num, prTimeLineForLastCommit)
	}
	return nil
}

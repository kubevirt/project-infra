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
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
	"kubevirt.io/project-infra/external-plugins/botreview/review"
	"sigs.k8s.io/prow/pkg/config"
	gitv2 "sigs.k8s.io/prow/pkg/git/v2"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/pluginhelp"
)

// HelpProvider construct the pluginhelp.PluginHelp for this plugin.
func HelpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The botreview plugin is used to automatically perform reviews of simple pull requests.`,
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/botreview",
		Description: "Trigger review of a PR.",
		Featured:    true,
		WhoCanUse:   "Project members",
		Examples:    []string{"/botreview"},
	})
	return pluginHelp, nil
}

// Server implements http.Handler. It validates incoming GitHub webhooks and
// then dispatches them to the appropriate plugins.
type Server struct {
	TokenGenerator func() []byte
	BotName        string

	GitClient gitv2.ClientFactory
	Ghc       github.Client

	Log *logrus.Entry

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
	//https://developer.github.com/webhooks/event-payloads/#pull_request
	case "pull_request":
		var pr github.PullRequestEvent
		if err := json.Unmarshal(payload, &pr); err != nil {
			return err
		}
		go func() {
			if err := s.handlePR(l, pr); err != nil {
				s.Log.WithError(err).WithFields(l.Data).Info("botreview failed.")
			}
		}()
	default:
		s.Log.WithFields(l.Data).Debugf("skipping event of type %q", eventType)
	}
	return nil
}

func (s *Server) handlePR(l *logrus.Entry, pr github.PullRequestEvent) error {
	action := pr.Action
	org := pr.Repo.Owner.Login
	repo := pr.Repo.Name
	num := pr.Number
	user := pr.Sender.Login

	return s.handlePullRequest(l, action, org, repo, num, user)
}

func (s *Server) handlePullRequest(l *logrus.Entry, action github.PullRequestEventAction, org string, repo string, num int, user string) error {
	switch action {
	case github.PullRequestActionOpened:
	case github.PullRequestActionEdited:
	case github.PullRequestActionReadyForReview:
	case github.PullRequestActionReopened:
		break
	default:
		l.Info("skipping review")
		return nil
	}
	prReviewOptions := review.PRReviewOptions{
		PullRequestNumber: num,
		Org:               org,
		Repo:              repo,
	}
	pullRequest, cloneDirectory, err := review.PreparePullRequestReview(s.GitClient, prReviewOptions, s.Ghc)
	if err != nil {
		logrus.WithError(err).Fatal("error preparing pull request for review")
	}
	err = os.Chdir(cloneDirectory)
	if err != nil {
		logrus.WithError(err).Fatal("error changing to directory")
	}

	reviewer := review.NewReviewer(l, action, org, repo, num, user, s.DryRun)
	reviewer.BaseSHA = pullRequest.Base.SHA
	botReviewResults, err := reviewer.ReviewLocalCode()
	if err != nil {
		return err
	}
	logrus.Infof("bot review results: %v", botReviewResults)
	if len(botReviewResults) == 0 {
		return nil
	}

	return reviewer.AttachReviewComments(botReviewResults, s.Ghc)
}

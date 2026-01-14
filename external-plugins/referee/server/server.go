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
	"html/template"
	"net/http"

	"github.com/sirupsen/logrus"
	"kubevirt.io/project-infra/external-plugins/referee/ghgraphql"
	"kubevirt.io/project-infra/external-plugins/referee/metrics"
	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/pjutil"
	"sigs.k8s.io/prow/pkg/pluginhelp"
)

const PluginName = "referee"
const DefaultMaximumNumberOfAllowedRetestComments = 5

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

	// MaximumNumberOfAllowedRetestComments defines the max number of allowed retests per commit
	// value defaults to DefaultMaximumNumberOfAllowedRetestComments
	MaximumNumberOfAllowedRetestComments int

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
		go func(ic github.IssueCommentEvent) {
			if err := s.handlePullRequestComment(ic); err != nil {
				s.Log.WithError(err).WithFields(l.Data).Errorf("%s failed.", PluginName)
			}
		}(ic)
	//https://developer.github.com/webhooks/event-payloads/#pull_request
	case "pull_request":
		var pr github.PullRequestEvent
		if err := json.Unmarshal(payload, &pr); err != nil {
			return err
		}
		go func(pr github.PullRequestEvent) {
			if err := s.handlePREvent(pr); err != nil {
				s.Log.WithError(err).WithFields(l.Data).Errorf("%s failed.", PluginName)
			}
		}(pr)
	default:
		s.Log.WithFields(l.Data).Debugf("skipping event of type %q", eventType)
	}
	return nil
}

func (s *Server) handlePREvent(pr github.PullRequestEvent) error {
	org := pr.Repo.Owner.Login
	repo := pr.Repo.Name
	num := pr.Number
	action := pr.Action

	pullRequestURL := fmt.Sprintf("https://github.com/%s/%s/pull/%d", org, repo, num)
	log := s.Log.WithField("pull_request_url", pullRequestURL)

	switch action {
	case github.PullRequestActionConvertedToDraft:
		// PRs that have been converted to draft are not tested thus they are not of interest for metrics
		metrics.DeleteForPullRequest(org, repo, num)
		return nil
	case github.PullRequestActionClosed:
		// PRs that have been merged or closed are not of interest for metrics
		metrics.DeleteForPullRequest(org, repo, num)
		return nil
	case github.PullRequestActionOpened:
	case github.PullRequestActionReopened:
	case github.PullRequestActionReadyForReview:
	case github.PullRequestActionSynchronize:
		// the above cases are the ones where we need to update the metrics
	default:
		// all other cases -> no metric action necessary
		log.Infof("skipping pull_request event action %s", action)
		return nil
	}
	return updateMetricsForPRs(s, org, repo, num, pullRequestURL, log)
}

func updateMetricsForPRs(s *Server, org string, repo string, num int, pullRequestURL string, log *logrus.Entry) error {
	prTimeLineForLastCommit, err := s.GHGraphQLClient.FetchPRTimeLineForLastCommit(org, repo, num)
	if err != nil {
		return fmt.Errorf("%s - failed to fetch number of retest comments: %w", pullRequestURL, err)
	}
	switch prTimeLineForLastCommit.NumberOfRetestComments {
	case 0:
		metrics.DeleteForPullRequest(org, repo, num)
	default:
		metrics.SetForPullRequest(org, repo, num, prTimeLineForLastCommit.NumberOfRetestComments)
	}
	log.Infof("updated metrics on PR")
	return nil
}

func (s *Server) handlePullRequestComment(ic github.IssueCommentEvent) error {
	org := ic.Repo.Owner.Login
	repo := ic.Repo.Name
	num := ic.Issue.Number
	action := ic.Action
	commentAuthor := ic.Comment.User.Login

	pullRequestURL := fmt.Sprintf("https://github.com/%s/%s/pull/%d", org, repo, num)
	log := s.Log.WithField("pull_request_url", pullRequestURL)

	switch action {
	case github.IssueCommentActionCreated:
	default:
		log.Debugf("skipping for action %s by %s", action, commentAuthor)
		return nil
	}

	if !ic.Issue.IsPullRequest() {
		log.Debugf("skipping since not a pull request")
		return nil
	}

	if !pjutil.RetestRe.MatchString(ic.Comment.Body) &&
		!pjutil.RetestRequiredRe.MatchString(ic.Comment.Body) &&
		!pjutil.TestAllRe.MatchString(ic.Comment.Body) {
		log.Debugf("skipping since comment didn't contain command triggering tests")
		return nil
	}

	// increase retest rate metric
	metrics.IncForRepository(org, repo)

	prTimeLineForLastCommit, err := s.GHGraphQLClient.FetchPRTimeLineForLastCommit(org, repo, num)
	if err != nil {
		return fmt.Errorf("%s - failed to fetch number of retest comments: %w", pullRequestURL, err)
	}

	// update per pr retest rate
	metrics.SetForPullRequest(org, repo, num, prTimeLineForLastCommit.NumberOfRetestComments)

	maxNumberOfAllowedRetestComments := s.maxNumberOfAllowedRetestComments()
	if prTimeLineForLastCommit.NumberOfRetestComments < maxNumberOfAllowedRetestComments {
		log.Debugf("skipping due to number of retest comments (%d) less than max allowed (%d)", prTimeLineForLastCommit.NumberOfRetestComments, maxNumberOfAllowedRetestComments)
		return nil
	}

	log.Warnf("excessive number of retest comments: %d", prTimeLineForLastCommit.NumberOfRetestComments)

	labels, err := s.GHGraphQLClient.FetchPRLabels(org, repo, num)
	if err != nil {
		return fmt.Errorf("%s - failed to fetch labels: %w", pullRequestURL, err)
	}
	if labels.IsHoldPresent {
		log.Infof("skipping due to hold present")
		return nil
	}

	if prTimeLineForLastCommit.WasHeld {
		for _, item := range prTimeLineForLastCommit.PRTimeLineItems {
			switch item.ItemType {
			case ghgraphql.HoldComment:
				if item.Item.Author.Login == s.BotName {
					log.Infof("skipping due to previous hold set by user %s", item.Item.Author.Login)
					return nil
				}
			default:
				continue
			}
		}
	}

	if !s.DryRun {
		prAuthor := ic.Issue.User.Login
		var output bytes.Buffer
		err := tooManyRetestsCommentTemplate.Execute(&output, TooManyRequestsData{
			Author: prAuthor,
			Team:   s.Team,
		})
		if err != nil {
			return fmt.Errorf("error while rendering comment template: %v", err)
		}
		err = s.GithubClient.CreateComment(org, repo, num, output.String())
		if err != nil {
			return fmt.Errorf("error while creating review comment: %v", err)
		}
	}
	return nil
}

func (s *Server) maxNumberOfAllowedRetestComments() int {
	if s.MaximumNumberOfAllowedRetestComments > 0 {
		return s.MaximumNumberOfAllowedRetestComments
	}
	return DefaultMaximumNumberOfAllowedRetestComments
}

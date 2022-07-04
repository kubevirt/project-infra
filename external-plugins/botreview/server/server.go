package server

import (
	"encoding/json"
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"kubevirt.io/project-infra/external-plugins/botreview/review"
	"net/http"
)

const pluginName = "botreview"

type issueEvent struct {
	github.IssueEvent `json:",inline"`
	Sender            github.User `json:"sender"`
}

type githubClient interface {
	AddLabel(org, repo string, number int, label string) error
	RemoveLabel(org, repo string, number int, label string) error
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	CreateComment(org, repo string, number int, comment string) error
	IsMember(org, user string) (bool, error)
}

// HelpProvider construct the pluginhelp.PluginHelp for this plugin.
func HelpProvider(_ []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: `The botreview plugin is used to automatically perform reviews of simple pull requests.`,
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/botreview",
		Description: "Mark a PR or issue as a release blocker.",
		Featured:    true,
		WhoCanUse:   "Project members",
		Examples:    []string{"/release-blocker release-3.9", "/release-blocker release-1.15"},
	})
	return pluginHelp, nil
}

// Server implements http.Handler. It validates incoming GitHub webhooks and
// then dispatches them to the appropriate plugins.
type Server struct {
	TokenGenerator func() []byte
	BotName        string

	// Used for unit testing
	Ghc githubClient
	Log *logrus.Entry
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

	// TODO: make dryRun configurable
	reviewer := review.NewReviewer(l, action, org, repo, num, user, true)
	botReviewResults, err := reviewer.ReviewLocalCode()
	if err != nil {
		return err
	}

	// TODO: casting will NOT work here
	err = reviewer.AttachReviewComments(botReviewResults, s.Ghc.(github.Client))
	if err != nil {
		return err
	}
	return nil
}

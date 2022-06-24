package server

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/go-diff/diff"
	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"kubevirt.io/project-infra/external-plugins/botreview/review"
	"net/http"
	"os/exec"
	"strings"
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
	withMessage := func(message string, args ...interface{}) string {
		return fmt.Sprintf("%s/%s#%d %s! <- %s: %s", org, repo, num, string(action), user, fmt.Sprintf(message, args))
	}
	infoF := func(message string, args ...interface{}) { l.Infof(withMessage(message, args)) }
	fatalF := func(message string, args ...interface{}) { l.Fatalf(withMessage(message, args)) }
	debugF := func(message string, args ...interface{}) { l.Debugf(withMessage(message, args)) }

	switch action {
	case github.PullRequestActionOpened:
	case github.PullRequestActionEdited:
	case github.PullRequestActionReadyForReview:
	case github.PullRequestActionReopened:
		break
	default:
		infoF("skipping review")
		return nil
	}

	infoF("preparing review")

	diffCommand := exec.Command("git", "diff", "..main")
	output, err := diffCommand.Output()
	if err != nil {
		fatalF("could not fetch diff output: %v", err)
	}

	multiFileDiffReader := diff.NewMultiFileDiffReader(strings.NewReader(string(output)))
	files, err := multiFileDiffReader.ReadAllFiles()
	if err != nil {
		fatalF("could not create diffs from output: %v", err)
	}

	types := review.GuessReviewTypes(files)
	debugF("review types: %v", types)
	if len(types) > 1 {
		infoF("doesn't look like a simple review, skipping")
		return nil
	}
	for _, reviewType := range types {
		result := reviewType.Review()
		l.Infof("%+v", result)
	}

	// TODO:

	return nil
}

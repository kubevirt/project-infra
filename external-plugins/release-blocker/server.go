package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"

	"k8s.io/test-infra/prow/config"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pluginhelp"
	"k8s.io/test-infra/prow/plugins"
)

const pluginName = "release-block"

var releaseBlockRe = regexp.MustCompile(`(?m)^(?:/releaseblock|/release-block)\s+(.+)$`)
var releaseBlockCancelRe = regexp.MustCompile(`(?m)^(?:/releaseblock\s+cancel|/release-block\s+cancel)\s+(.+)$`)

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
		Description: `The release-block plugin is used to signal an issue or PR must be resolved before the next release is made.`,
	}
	pluginHelp.AddCommand(pluginhelp.Command{
		Usage:       "/release-block [branch]",
		Description: "Mark a PR or issue as a release blocker.",
		Featured:    true,
		WhoCanUse:   "Project members",
		Examples:    []string{"/release-block release-3.9", "/release-block release-1.15"},
	})
	return pluginHelp, nil
}

// Server implements http.Handler. It validates incoming GitHub webhooks and
// then dispatches them to the appropriate plugins.
type Server struct {
	TokenGenerator func() []byte
	BotName        string

	// Used for unit testing
	push func(newBranch string) error
	GHC  githubClient
	Log  *logrus.Entry
}

// ServeHTTP validates an incoming webhook and puts it into the event channel.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.TokenGenerator)
	if !ok {
		return
	}
	fmt.Fprint(w, "Event received. Have a nice day.")

	if err := s.handleEvent(eventType, eventGUID, payload); err != nil {
		logrus.WithError(err).Error("Error parsing event.")
	}
}

func hasLabel(ghc githubClient, label string, org string, repo string, num int) (bool, error) {
	labels, err := ghc.GetIssueLabels(org, repo, num)
	if err != nil {
		return false, fmt.Errorf("failed to get the labels on %s/%s#%d: %v", org, repo, num, err)
	}

	hasLabel := github.HasLabel(label, labels)

	return hasLabel, nil
}

func canLabel(ghc githubClient, org string, commentAuthor string) (bool, error) {
	// only members can add blocking label.
	ok, err := ghc.IsMember(org, commentAuthor)
	if err != nil {
		return false, err
	}

	return ok, nil
}

func (s *Server) handleEvent(eventType, eventGUID string, payload []byte) error {
	l := logrus.WithFields(
		logrus.Fields{
			"event-type":     eventType,
			github.EventGUID: eventGUID,
		},
	)
	switch eventType {
	case "issue_comment":
		var ic github.IssueCommentEvent
		if err := json.Unmarshal(payload, &ic); err != nil {
			return err
		}
		go func() {
			if err := s.handleIssueComment(l, ic); err != nil {
				s.Log.WithError(err).WithFields(l.Data).Info("release-block issue comment failed.")
			}
		}()
	/*
		// TODO
		case "issue":
			// https://developer.github.com/webhooks/event-payloads/#issues
			// look for label/unlabel events, make sure only bot sets blocker labels
		// TODO
		case "pull_request":
			//https://developer.github.com/webhooks/event-payloads/#pull_request
			// look for label/unlabel events, make sure only bot sets blocker labels
			var pr github.PullRequestEvent
			if err := json.Unmarshal(payload, &pr); err != nil {
				return err
			}
				go func() {
					if err := s.handlePullRequest(l, pr); err != nil {
						s.Log.WithError(err).WithFields(l.Data).Info("release-block failed.")
					}
				}()
	*/
	default:
		logrus.WithFields(l.Data).Debugf("skipping event of type %q", eventType)
	}
	return nil
}

func (s *Server) handleIssueComment(l *logrus.Entry, ic github.IssueCommentEvent) error {
	// we're only interested in new comments
	if ic.Action != github.IssueCommentActionCreated {
		return nil
	}

	org := ic.Repo.Owner.Login
	repo := ic.Repo.Name
	num := ic.Issue.Number
	commentAuthor := ic.Comment.User.Login

	needsLabel := true
	targetBranch := ""

	l = l.WithFields(logrus.Fields{
		github.OrgLogField:  org,
		github.RepoLogField: repo,
		github.PrLogField:   num,
	})

	cancelMatches := releaseBlockCancelRe.FindAllStringSubmatch(ic.Comment.Body, -1)
	matches := releaseBlockRe.FindAllStringSubmatch(ic.Comment.Body, -1)

	if len(cancelMatches) == 1 && len(cancelMatches[0]) == 2 {
		needsLabel = false
		targetBranch = strings.TrimSpace(cancelMatches[0][1])
	} else if len(matches) == 1 && len(matches[0]) == 2 {
		needsLabel = true
		targetBranch = strings.TrimSpace(matches[0][1])
	} else {
		// no matches
		return nil
	}

	// validate the user is allowed to block or unblock
	ok, err := canLabel(s.GHC, org, commentAuthor)
	if err != nil {
		return err
	}

	// not authorized.
	if !ok {
		resp := fmt.Sprintf("only [%s](https://github.com/orgs/%s/people) org members may request release block label", org, org)
		s.Log.WithFields(l.Data).Info(resp)
		return s.GHC.CreateComment(org, repo, num, plugins.FormatICResponse(ic.Comment, resp))
	}

	s.Log.WithFields(l.Data).
		WithField("requestor", ic.Comment.User.Login).
		WithField("target_branch", targetBranch).
		Debug("release-blocker request.")

	// TODO validate branch exists
	label := fmt.Sprintf("release-block/%s", targetBranch)

	hasLabel, err := hasLabel(s.GHC, label, org, repo, num)
	if err != nil {
		s.Log.WithFields(l.Data).WithError(err)
		return err
	}

	if needsLabel && !hasLabel {
		err := s.GHC.AddLabel(org, repo, num, label)
		if err != nil {
			s.Log.WithFields(l.Data).WithError(err)
			return err
		}
	} else if !needsLabel && hasLabel {
		err := s.GHC.RemoveLabel(org, repo, num, label)
		if err != nil {
			s.Log.WithFields(l.Data).WithError(err)
			return err
		}
	}

	return nil
}

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/config/secret"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/pluginhelp/externalplugins"
)

const pluginName = "docs-pr-checker"

var (
	// Matches a docs-pr code block, capturing everything between the opening and closing triple backticks.
	// Handles optional whitespace/newlines and any content inside.
	docsPRRegex = regexp.MustCompile(`(?is)```docs-pr\s*\n(.*?)\n?```)`)
)

const (
	labelDocsPRRequired = "do-not-merge/docs-pr-required"
	labelDocsPR         = "docs-pr"
	labelDocsPRNone     = "docs-pr-none"
)

// HelpProvider constructs the PluginHelp for this plugin.
func HelpProvider(config []byte) (*externalplugins.PluginHelp, error) {
	pluginHelp := &externalplugins.PluginHelp{
		Description: "The docs-pr-checker plugin ensures that PRs include documentation updates when required.",
	}
	pluginHelp.AddCommand(externalplugins.PluginCommand{
		Usage:       "/docs-pr <PR-number|NONE>",
		Description: "Updates the docs-pr status of a pull request.",
		Featured:    false,
		WhoCanUse:   "Anyone can trigger this command on a PR.",
		Examples:    []string{"/docs-pr #123", "/docs-pr NONE"},
	})
	return pluginHelp, nil
}

// Server implements http.Handler. It validates incoming GitHub webhooks and
// then dispatches them to the appropriate plugins.
type Server struct {
	tokenGenerator func() []byte
	botName        string
	ghc            github.Client
	log            *logrus.Entry
}

// ServeHTTP validates an incoming webhook and puts it into the event channel.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	eventType, eventGUID, payload, ok, _ := github.ValidateWebhook(w, r, s.tokenGenerator)
	if !ok {
		return
	}
	fmt.Fprint(w, "Event received. Have a nice day.")

	if err := s.handleEvent(eventType, eventGUID, payload); err != nil {
		logrus.WithError(err).Error("Error parsing event.")
	}
}

func (s *Server) handleEvent(eventType, eventGUID string, payload []byte) error {
	l := s.log.WithFields(
		logrus.Fields{
			"event-type":     eventType,
			github.EventGUID: eventGUID,
		},
	)
	var handler func() error
	switch eventType {
	case "pull_request":
		var pre github.PullRequestEvent
		if err := json.Unmarshal(payload, &pre); err != nil {
			return err
		}
		handler = func() error { return s.handlePullRequestEvent(l, &pre) }
	case "issue_comment":
		var ice github.IssueCommentEvent
		if err := json.Unmarshal(payload, &ice); err != nil {
			return err
		}
		handler = func() error { return s.handleIssueCommentEvent(l, &ice) }
	default:
		l.Debugf("skipping event of type %q", eventType)
		return nil
	}
	go func() {
		if err := handler(); err != nil {
			l.WithError(err).Info("Error handling event.")
		}
	}()
	return nil
}

func (s *Server) handlePullRequestEvent(l *logrus.Entry, pre *github.PullRequestEvent) error {
	// Only process opened, edited, and synchronize events
	if pre.Action != github.PullRequestActionOpened &&
		pre.Action != github.PullRequestActionEdited &&
		pre.Action != github.PullRequestActionSynchronize {
		return nil
	}

	return s.processPR(l, pre.Repo.Owner.Login, pre.Repo.Name, &pre.PullRequest)
}

func (s *Server) handleIssueCommentEvent(l *logrus.Entry, ice *github.IssueCommentEvent) error {
	// Only process comments on PRs
	if !ice.Issue.IsPullRequest() {
		return nil
	}

	// Check if comment is a docs-pr command
	if !strings.HasPrefix(ice.Comment.Body, "/docs-pr") {
		return nil
	}

	// Get the PR details
	pr, err := s.ghc.GetPullRequest(ice.Repo.Owner.Login, ice.Repo.Name, ice.Issue.Number)
	if err != nil {
		return fmt.Errorf("failed to get pull request: %w", err)
	}

	// Process the command
	return s.processDocsPRCommand(l, ice.Repo.Owner.Login, ice.Repo.Name, pr, ice.Comment.Body)
}

func (s *Server) processPR(l *logrus.Entry, org, repo string, pr *github.PullRequest) error {
	return s.checkAndUpdateDocsPRStatus(l, org, repo, pr)
}

func (s *Server) processDocsPRCommand(l *logrus.Entry, org, repo string, pr *github.PullRequest, comment string) error {
	// Parse the command
	parts := strings.Fields(comment)
	if len(parts) < 2 {
		return nil // Invalid command, ignore
	}

	docsPRValue := strings.Join(parts[1:], " ")
	
	// Update PR body with the new docs-pr value
	newBody := s.updateDocsPRInBody(pr.Body, docsPRValue)
	
	// Update the PR description
	update := github.PullRequest{
		Body: &newBody,
	}
	
	if err := s.ghc.EditPullRequest(org, repo, pr.Number, &update); err != nil {
		return fmt.Errorf("failed to update PR body: %w", err)
	}

	// Update the labels based on the new value
	updatedPR := *pr
	updatedPR.Body = newBody
	
	return s.checkAndUpdateDocsPRStatus(l, org, repo, &updatedPR)
}

func (s *Server) updateDocsPRInBody(body, newValue string) string {
	// Remove any existing docs-pr blocks
	cleanBody := docsPRRegex.ReplaceAllString(body, "")
	// Trim any trailing whitespace after removal
	cleanBody = strings.TrimSpace(cleanBody)
	// Append a single docs-pr block at the end
	return cleanBody + fmt.Sprintf("\n\n```docs-pr\n%s\n```", newValue)
}

func (s *Server) checkAndUpdateDocsPRStatus(l *logrus.Entry, org, repo string, pr *github.PullRequest) error {
	current, err := s.ghc.GetIssueLabels(org, repo, pr.Number)
	if err != nil {
		return fmt.Errorf("failed to get current labels: %w", err)
	}

	// inline extractDocsPRValue
	var val string
	if m := docsPRRegex.FindStringSubmatch(pr.Body); len(m) > 1 {
		val = strings.TrimSpace(m[1])
	}

	// compute desired state for each label
	desired := map[string]bool{
		labelDocsPRRequired: val == "",
		labelDocsPRNone:     strings.EqualFold(val, "NONE"),
		labelDocsPR:         val != "" && !strings.EqualFold(val, "NONE"),
	}

	// sync labels in one pass
	for _, target := range []string{labelDocsPRRequired, labelDocsPR, labelDocsPRNone} {
		present := false
		for _, lbl := range current {
			if lbl.Name == target {
				present = true
				break
			}
		}
		if desired[target] && !present {
			if err := s.ghc.AddLabel(org, repo, pr.Number, target); err != nil {
				l.WithError(err).Warnf("failed to add label %s", target)
			} else {
				l.Infof("added label %s", target)
			}
		}
		if !desired[target] && present {
			if err := s.ghc.RemoveLabel(org, repo, pr.Number, target); err != nil {
				l.WithError(err).Warnf("failed to remove label %s", target)
			} else {
				l.Infof("removed label %s", target)
			}
		}
	}
	return nil
}



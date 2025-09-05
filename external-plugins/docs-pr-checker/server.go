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
	docsPRRegex = regexp.MustCompile(`(?i)```docs-pr\s*\n([^\n\`]*)\n`)
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
	switch eventType {
	case "pull_request":
		var pre github.PullRequestEvent
		if err := json.Unmarshal(payload, &pre); err != nil {
			return err
		}
		go func() {
			if err := s.handlePullRequestEvent(l, &pre); err != nil {
				l.WithError(err).Info("Error handling pull request event.")
			}
		}()
	case "issue_comment":
		var ice github.IssueCommentEvent
		if err := json.Unmarshal(payload, &ice); err != nil {
			return err
		}
		go func() {
			if err := s.handleIssueCommentEvent(l, &ice); err != nil {
				l.WithError(err).Info("Error handling issue comment event.")
			}
		}()
	default:
		l.Debugf("skipping event of type %q", eventType)
	}
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
	// If docs-pr section exists, replace it
	if docsPRRegex.MatchString(body) {
		return docsPRRegex.ReplaceAllString(body, fmt.Sprintf("```docs-pr\n%s\n", newValue))
	}
	
	// If no docs-pr section exists, add it at the end
	return body + fmt.Sprintf("\n\n```docs-pr\n%s\n```", newValue)
}

func (s *Server) checkAndUpdateDocsPRStatus(l *logrus.Entry, org, repo string, pr *github.PullRequest) error {
	currentLabels, err := s.ghc.GetIssueLabels(org, repo, pr.Number)
	if err != nil {
		return fmt.Errorf("failed to get current labels: %w", err)
	}

	// Extract docs-pr value from PR body
	docsPRValue := s.extractDocsPRValue(pr.Body)
	
	// Determine what labels should be set
	var labelsToAdd, labelsToRemove []string
	
	if docsPRValue == "" {
		// No docs-pr field or empty - require docs PR
		labelsToAdd = []string{labelDocsPRRequired}
		labelsToRemove = []string{labelDocsPR, labelDocsPRNone}
	} else if strings.ToUpper(strings.TrimSpace(docsPRValue)) == "NONE" {
		// Docs update not needed
		labelsToAdd = []string{labelDocsPRNone}
		labelsToRemove = []string{labelDocsPRRequired, labelDocsPR}
	} else {
		// Docs PR provided
		labelsToAdd = []string{labelDocsPR}
		labelsToRemove = []string{labelDocsPRRequired, labelDocsPRNone}
	}

	// Remove labels that should not be present
	for _, label := range labelsToRemove {
		if s.hasLabel(currentLabels, label) {
			if err := s.ghc.RemoveLabel(org, repo, pr.Number, label); err != nil {
				l.WithError(err).Warnf("failed to remove label %s", label)
			} else {
				l.Infof("removed label %s", label)
			}
		}
	}

	// Add labels that should be present
	for _, label := range labelsToAdd {
		if !s.hasLabel(currentLabels, label) {
			if err := s.ghc.AddLabel(org, repo, pr.Number, label); err != nil {
				l.WithError(err).Warnf("failed to add label %s", label)
			} else {
				l.Infof("added label %s", label)
			}
		}
	}

	return nil
}

func (s *Server) extractDocsPRValue(body string) string {
	matches := docsPRRegex.FindStringSubmatch(body)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

func (s *Server) hasLabel(labels []github.Label, labelName string) bool {
	for _, label := range labels {
		if label.Name == labelName {
			return true
		}
	}
	return false
}


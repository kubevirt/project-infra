package main

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

const pluginName = "docs-pr-checker"

var (
	// Matches a docs-pr code block, capturing everything between the opening and closing triple backticks.
	// Handles optional whitespace/newlines and any content inside.
	docsPRRegex = regexp.MustCompile("(?is)" + "```" + `docs-pr\s*\n(.*?)\n?` + "```")

	// isValidDocsPRReference matches valid references, like #123, repo#123, org/repo#123, or NONE (case-insensitive).
	isValidDocsPRReference = regexp.MustCompile(`(?i)^(NONE|([a-zA-Z0-9-]+\/[a-zA-Z0-9-]+)?#\d+)$`)
)

const (
	labelDocsPRRequired = "do-not-merge/docs-pr-required"
	labelDocsPR         = "docs-pr"
	labelDocsPRNone     = "docs-pr-none"
)

// HelpProvider constructs the PluginHelp for this plugin.
func HelpProvider(enabledRepos []config.OrgRepo) (*pluginhelp.PluginHelp, error) {
	pluginHelp := &pluginhelp.PluginHelp{
		Description: "The docs-pr-checker plugin ensures that PRs include documentation updates when required.",
	}
	pluginHelp.AddCommand(pluginhelp.Command{
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
	ghc            prClient
	log            *logrus.Entry
}

type prClient interface {
	GetPullRequest(org, repo string, number int) (*github.PullRequest, error)
	EditPullRequest(org, repo string, number int, pr *github.PullRequest) (*github.PullRequest, error)
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	AddLabel(org, repo string, number int, label string) error
	RemoveLabel(org, repo string, number int, label string) error
	CreateComment(org, repo string, number int, comment string) error
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
		Body: newBody,
	}

	if _, err := s.ghc.EditPullRequest(org, repo, pr.Number, &update); err != nil {
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
	// Remove extra newlines
	cleanBody = strings.ReplaceAll(cleanBody, "\n\n", "\n")
	// Append a single docs-pr block at the end
	result := cleanBody + fmt.Sprintf("\n```docs-pr\n%s\n```", newValue)
	s.log.Infof("updateDocsPRInBody: body=%q, newValue=%q, result=%q", body, newValue, result)
	return result
}

func (s *Server) extractDocsPRValue(body string) string {
	if m := docsPRRegex.FindStringSubmatch(body); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func (s *Server) hasLabel(labels []github.Label, label string) bool {
	for _, l := range labels {
		if l.Name == label {
			return true
		}
	}
	return false
}

func (s *Server) checkAndUpdateDocsPRStatus(l *logrus.Entry, org, repo string, pr *github.PullRequest) error {
	currentIssueLabels, err := s.ghc.GetIssueLabels(org, repo, pr.Number)
	if err != nil {
		return fmt.Errorf("failed to get current labels: %w", err)
	}

	val := s.extractDocsPRValue(pr.Body)
	isValid := val != "" && isValidDocsPRReference.MatchString(val)

	// compute desired state for each label
	desiredLabels := map[string]bool{
		labelDocsPRRequired: !isValid,
		labelDocsPRNone:     isValid && strings.EqualFold(val, "NONE"),
		labelDocsPR:         isValid && !strings.EqualFold(val, "NONE"),
	}

	// Add a comment if the docs-pr value is invalid
	if val != "" && !isValid {
		comment := fmt.Sprintf("Invalid `docs-pr` value: `%s`. Please use `NONE` or a valid PR reference (e.g., `#123`, `repo#123`, `org/repo#123`).", val)
		if err := s.ghc.CreateComment(org, repo, pr.Number, comment); err != nil {
			l.WithError(err).Warn("failed to create comment")
		}
	}

	// sync labels in one pass
	for _, target := range []string{labelDocsPRRequired, labelDocsPR, labelDocsPRNone} {
		present := s.hasLabel(currentIssueLabels, target)
		if desiredLabels[target] && !present {
			if err := s.ghc.AddLabel(org, repo, pr.Number, target); err != nil {
				l.WithError(err).Warnf("failed to add label %s", target)
			} else {
				l.Infof("added label %s", target)
			}
		}
		if !desiredLabels[target] && present {
			if err := s.ghc.RemoveLabel(org, repo, pr.Number, target); err != nil {
				l.WithError(err).Warnf("failed to remove label %s", target)
			} else {
				l.Infof("removed label %s", target)
			}
		}
	}
	return nil
}

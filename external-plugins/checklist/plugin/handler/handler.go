package handler

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	gitv2 "k8s.io/test-infra/prow/git/v2"
	"k8s.io/test-infra/prow/github"
)

var log *logrus.Logger

const IncompleteChecklistLabel = "do-not-merge/incomplete-checklist"

func init() {
	log = logrus.New()
	log.SetOutput(os.Stdout)
}

type GitHubEvent struct {
	Type    string
	GUID    string
	Payload []byte
}

type githubClientInterface interface {
	GetPullRequest(string, string, int) (*github.PullRequest, error)
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
	AddLabel(org, repo string, number int, label string) error
	RemoveLabel(org, repo string, number int, label string) error
}

type GitHubEventsHandler struct {
	eventsChan <-chan *GitHubEvent
	logger     *logrus.Logger
	ghClient   githubClientInterface
}

func NewGitHubEventsHandler(
	eventsChan <-chan *GitHubEvent,
	logger *logrus.Logger,
	ghClient githubClientInterface) *GitHubEventsHandler {

	return &GitHubEventsHandler{
		eventsChan: eventsChan,
		logger:     logger,
		ghClient:   ghClient,
	}
}

func (h *GitHubEventsHandler) Handle(incomingEvent *GitHubEvent) {
	log.Infoln("GitHub events handler started")
	eventLog := log.WithField("event-guid", incomingEvent.GUID)
	switch incomingEvent.Type {
	case "pull_request":
		eventLog.Infoln("Handling pull request event")
		var event github.PullRequestEvent
		if err := json.Unmarshal(incomingEvent.Payload, &event); err != nil {
			eventLog.WithError(err).Error("Could not unmarshal event.")
			return
		}
		h.handlePullRequestEvent(eventLog, &event)
	default:
		return
	}
}

func (h *GitHubEventsHandler) handlePullRequestEvent(log *logrus.Entry, event *github.PullRequestEvent) {
	log.Infof("Handling updated pull request: %s [%d]", event.Repo.FullName, event.PullRequest.Number)

	if !h.shouldActOnPREvent(event) {
		return
	}

	org, repo, err := gitv2.OrgRepo(event.Repo.FullName)
	if err != nil {
		log.WithError(err).Errorf("Could not get org/repo from the event")
		return
	}

	pr, err := h.ghClient.GetPullRequest(org, repo, event.PullRequest.Number)
	if err != nil {
		log.WithError(err).Errorf("Could not get PR number %d", event.PullRequest.Number)
		return
	}

	hasLabel, err := h.hasChecklistLabel(org, repo, event.PullRequest.Number)
	if err != nil {
		log.WithError(err).Errorf("Could not get PR labels %d", event.PullRequest.Number)
		return
	}

	if checkPRBody(pr) {
		if hasLabel {
			err = h.ghClient.RemoveLabel(org, repo, event.PullRequest.Number, IncompleteChecklistLabel)
			if err != nil {
				log.WithError(err).Errorf("Could not remove label from PR %d", event.PullRequest.Number)
				return
			}
		}
	} else {
		if !hasLabel {
			err = h.ghClient.AddLabel(org, repo, event.PullRequest.Number, IncompleteChecklistLabel)
			if err != nil {
				log.WithError(err).Errorf("Could not add label to PR %d", event.PullRequest.Number)
				return
			}
		}
	}
}

func (h *GitHubEventsHandler) shouldActOnPREvent(event *github.PullRequestEvent) bool {
	return event.Action == github.PullRequestActionOpened || event.Action == github.PullRequestActionEdited
}

func checkPRBody(pr *github.PullRequest) bool {
	list := []string{"Design", "PR", "Code", "Refactor", "Upgrade", "Testing", "Documentation", "Community"}
	for _, item := range list {
		item = "- [ ] " + item + ":"
		if strings.Contains(pr.Body, item) {
			return false
		}
	}

	return true
}

func (h *GitHubEventsHandler) hasChecklistLabel(org, repo string, prNum int) (bool, error) {
	l, err := h.ghClient.GetIssueLabels(org, repo, prNum)
	if err != nil {
		log.WithError(err).Errorf("Could not get PR labels")
		return false, err
	}

	return github.HasLabel(IncompleteChecklistLabel, l), nil
}

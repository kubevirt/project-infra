package handler

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	prowapi "sigs.k8s.io/prow/pkg/apis/prowjobs/v1"
	prowv1 "sigs.k8s.io/prow/pkg/client/clientset/versioned/typed/prowjobs/v1"
	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/pjutil"
)

// Represents a GitHub webhook event
type GitHubEvent struct {
	Type    string
	GUID    string
	Payload []byte
}

// GitHub client interface
// Interface defines the methods the handler needs to interact with the GitHub API
type githubClient interface {
	GetPullRequestChanges(org, repo string, number int) ([]github.PullRequestChange, error)
}

// GitHub events handler
type GitHubEventsHandler struct {
	eventsChan    <-chan *GitHubEvent
	logger        *logrus.Logger
	prowJobClient prowv1.ProwJobInterface
	githubClient  githubClient
	jobsNamespace string
	dryrun        bool
}

// Creating a new GitHubEventsHandler
func NewGitHubEventsHandler(
	eventsChan <-chan *GitHubEvent,
	logger *logrus.Logger,
	prowJobClient prowv1.ProwJobInterface,
	githubClient githubClient,
	jobsNamespace string,
	dryrun bool,
) *GitHubEventsHandler {
	return &GitHubEventsHandler{
		eventsChan:    eventsChan,
		logger:        logger,
		prowJobClient: prowJobClient,
		githubClient:  githubClient,
		jobsNamespace: jobsNamespace,
		dryrun:        dryrun,
	}
}

func (h *GitHubEventsHandler) Handle(incomingEvent *GitHubEvent) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Warnf("Recovered during handling of event %s: %s", incomingEvent.GUID, r)
		}
	}()
	eventLog := h.logger.WithField("event-guid", incomingEvent.GUID)
	switch incomingEvent.Type {
	case "pull_request":
		eventLog.Infof("Handling pull request event")
		var event github.PullRequestEvent
		if err := json.Unmarshal(incomingEvent.Payload, &event); err != nil {
			eventLog.WithError(err).Error("Could not unmarshal event")
			return
		}
		h.handlePullRequestEvent(eventLog, &event)
	default:
		eventLog.Infof("Dropping irrelevant event type: %s", incomingEvent.Type)
	}
}

// Extracting file names
func extractFilenames(changes []github.PullRequestChange) []string {
	filenames := make([]string, len(changes))
	for i, change := range changes {
		filenames[i] = change.Filename
	}
	return filenames
}

// Detecting Go file changes in the files list
func detectGoFileChanges(files []string) bool {
	for _, file := range files {
		inCoverageDir := strings.HasPrefix(file, "external-plugins/") ||
			strings.HasPrefix(file, "releng/") ||
			strings.HasPrefix(file, "robots/") ||
			strings.HasPrefix(file, "pkg/")

		isGoFile := strings.HasSuffix(file, ".go")

		if inCoverageDir && isGoFile {
			return true
		}
	}
	return false
}

// Checking if the event should trigger the coverage plugin
func actOnPrEvent(event *github.PullRequestEvent) bool {
	return event.Action == github.PullRequestActionOpened ||
		event.Action == github.PullRequestActionSynchronize
}

// Generating a coverage job for a pull request
func (h *GitHubEventsHandler) generateCoverageJob(
	pr *github.PullRequest, eventGUID string) prowapi.ProwJob {
	presubmit := config.Presubmit{
		JobBase: config.JobBase{
			Name:    "coverage-auto",
			Cluster: "kubevirt-prow-control-plane",
			Labels: map[string]string{
				"coverage-plugin": "true",
			},
			Spec: &corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: "quay.io/kubevirtci/golang:v20251218-e7a7fc9",
						Command: []string{
							"/usr/local/bin/runner.sh",
							"/bin/sh",
							"-ce",
						},
						Args: []string{
							"make coverage",
						},
						Env: []corev1.EnvVar{
							{
								Name:  "GO_MOD_PATH",
								Value: "go.mod",
							},
						},
					},
				},
			},
			Namespace: &h.jobsNamespace,
		},
	}
	// Creating a new ProwJob using the pjutil package
	return pjutil.NewPresubmit(*pr, pr.Base.SHA, presubmit, eventGUID, nil)
}

// Retrieving changed files
func (h *GitHubEventsHandler) getPullRequestChanges(pr *github.PullRequest) ([]github.PullRequestChange, error) {
	return h.githubClient.GetPullRequestChanges(pr.Base.Repo.Owner.Login, pr.Base.Repo.Name, pr.Number)
}

// Handling a pull request event
func (h *GitHubEventsHandler) handlePullRequestEvent(log *logrus.Entry, prEvent *github.PullRequestEvent) {
	if !actOnPrEvent(prEvent) {
		log.Infof("Skipping PR event with action: %s", prEvent.Action)
		return
	}

	changes, err := h.getPullRequestChanges(&prEvent.PullRequest)
	if err != nil {
		log.WithError(err).Error("Failed to get pull request changes")
		return
	}

	if !detectGoFileChanges(extractFilenames(changes)) {
		log.Info("No Go file changes detected, skipping coverage job")
		return
	}

	eventGUID := log.Data["event-guid"].(string)
	job := h.generateCoverageJob(&prEvent.PullRequest, eventGUID)

	if h.dryrun {
		log.Infof("Dry-run: would create coverage job for PR #%d", prEvent.PullRequest.Number)
		return
	}

	if _, err := h.prowJobClient.Create(context.Background(), &job, metav1.CreateOptions{}); err != nil {
		log.WithError(err).Error("Failed to create coverage ProwJob")
		return
	}

	log.Infof("Created coverage job for PR #%d", prEvent.PullRequest.Number)
}

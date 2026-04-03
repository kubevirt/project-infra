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

// GitHubEvent represents a GitHub webhook event.
type GitHubEvent struct {
	Type    string
	GUID    string
	Payload []byte
}

// githubClient defines the methods the handler needs to interact with the GitHub API.
type githubClient interface {
	GetPullRequestChanges(org, repo string, number int) ([]github.PullRequestChange, error)
}

// GitHubEventsHandler handles incoming GitHub webhook events and creates coverage ProwJobs.
type GitHubEventsHandler struct {
	logger        *logrus.Logger
	prowJobClient prowv1.ProwJobInterface
	githubClient  githubClient
	jobsNamespace string
	dryrun        bool
}

// NewGitHubEventsHandler creates and returns a new GitHubEventsHandler.
func NewGitHubEventsHandler(
	logger *logrus.Logger,
	prowJobClient prowv1.ProwJobInterface,
	githubClient githubClient,
	jobsNamespace string,
	dryrun bool,
) *GitHubEventsHandler {
	return &GitHubEventsHandler{
		logger:        logger,
		prowJobClient: prowJobClient,
		githubClient:  githubClient,
		jobsNamespace: jobsNamespace,
		dryrun:        dryrun,
	}
}

// Handle processes an incoming GitHub webhook event.
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
		eventLog.Debugf("Dropping irrelevant event type: %s", incomingEvent.Type)
	}
}

// extractFilenames returns the filenames from a list of pull request changes.
func extractFilenames(changes []github.PullRequestChange) []string {
	var filenames []string
	for _, change := range changes {
		filenames = append(filenames, change.Filename)
	}
	return filenames
}

// detectGoFileChanges reports whether any of the given files is a Go source file.
func detectGoFileChanges(files []string) bool {
	for _, file := range files {
		isGoFile := strings.HasSuffix(file, ".go")
		if isGoFile {
			return true
		}
	}
	return false
}

// shouldActOnPREvent reports whether the given action should trigger the coverage plugin.
func shouldActOnPREvent(action string) bool {
	return action == string(github.PullRequestActionOpened) ||
		action == string(github.PullRequestActionSynchronize)
}

// generateCoverageJob creates a ProwJob for running coverage on the given pull request.
func (h *GitHubEventsHandler) generateCoverageJob(
	pr *github.PullRequest, eventGUID string) prowapi.ProwJob {
	decorate := true
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
						Image: "quay.io/kubevirtci/covreport:latest",
						Command: []string{
							"/usr/local/bin/entrypoint.sh",
							"/bin/sh",
							"-ce",
						},
						Args: []string{
							"go test ./... -coverprofile=/tmp/coverage.out && " +
								"covreport -i /tmp/coverage.out -o ${ARTIFACTS}/coverage.html",
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
			UtilityConfig: config.UtilityConfig{
				Decorate: &decorate,
			},
		},
		Reporter: config.Reporter{
			Context: "coverage-auto",
			SkipReport: true,
		},
	}
	return pjutil.NewPresubmit(*pr, pr.Base.SHA, presubmit, eventGUID, nil)
}

// getPullRequestChanges returns the files changed in the given pull request.
func (h *GitHubEventsHandler) getPullRequestChanges(pr *github.PullRequest) ([]github.PullRequestChange, error) {
	return h.githubClient.GetPullRequestChanges(pr.Base.Repo.Owner.Login, pr.Base.Repo.Name, pr.Number)
}

// handlePullRequestEvent processes a pull request event, creating a coverage ProwJob if Go files were changed.
func (h *GitHubEventsHandler) handlePullRequestEvent(log *logrus.Entry, prEvent *github.PullRequestEvent) {
	if !shouldActOnPREvent(string(prEvent.Action)) {
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

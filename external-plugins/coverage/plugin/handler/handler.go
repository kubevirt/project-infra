// TODO: Implement coverage detection and job generation

package handler

import (
	"strings"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
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

// Handles GitHub webhook events
type GitHubEventsHandler struct {
	eventsChan    <-chan *GitHubEvent
	logger        *logrus.Logger
	prowJobClient prowv1.ProwJobInterface
	githubClient  github.Client
	jobsNamespace string
	dryrun        bool
}

// Creates a new GitHubEventsHandler
func NewGitHubEventsHandler(
	eventsChan <-chan *GitHubEvent,
	logger *logrus.Logger,
	prowJobClient prowv1.ProwJobInterface,
	githubClient github.Client,
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

// Checks for Go file changes in the files list
func detectGoFileChanges(files []string) bool {
	for _, file := range files {
		inCoverageDir := strings.HasPrefix(file, "external-plugins/") ||
			strings.HasPrefix(file, "releng/") ||
			strings.HasPrefix(file, "robots/") ||
			strings.HasPrefix(file, "pkg/")

		isGoFile := strings.HasSuffix(file, ".go") ||
			strings.HasSuffix(file, "go.mod") ||
			strings.HasSuffix(file, "go.sum")

		if inCoverageDir && isGoFile {
			return true
		}
	}
	return false
}

// Determines if the event should trigger the coverage plugin
func actOnPrEvent(event *github.PullRequestEvent) bool {
	return event.Action == github.PullRequestActionOpened ||
		event.Action == github.PullRequestActionSynchronize
}

// Generates a coverage job for a pull request
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
	// Create a new ProwJob using the pjutil package
	return pjutil.NewPresubmit(*pr, pr.Base.SHA, presubmit, eventGUID, nil)
}

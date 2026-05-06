package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	prowapi "sigs.k8s.io/prow/pkg/apis/prowjobs/v1"
	prowv1 "sigs.k8s.io/prow/pkg/client/clientset/versioned/typed/prowjobs/v1"
	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/pjutil"
)

type UtilityImagesConfig struct {
	CloneRefs  string `yaml:"cloneRefs"`
	InitUpload string `yaml:"initUpload"`
	Entrypoint string `yaml:"entrypoint"`
	Sidecar    string `yaml:"sidecar"`
}

type GCSConfig struct {
	Bucket            string `yaml:"bucket"`
	PathStrategy      string `yaml:"pathStrategy"`
	CredentialsSecret string `yaml:"credentialsSecret"`
}

type JobConfig struct {
	Namespace          string              `yaml:"namespace"`
	Image              string              `yaml:"image"`
	Cluster            string              `yaml:"cluster"`
	TestPackages       string              `yaml:"testPackages"`
	Env                map[string]string   `yaml:"env"`
	TimeoutMinutes     int                 `yaml:"timeoutMinutes"`
	GracePeriodSeconds int                 `yaml:"gracePeriodSeconds"`
	UtilityImages      UtilityImagesConfig `yaml:"utilityImages"`
	GCS                GCSConfig           `yaml:"gcs"`
}

func LoadJobConfig(path string) (*JobConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	cfg := &JobConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	if cfg.Namespace == "" {
		return nil, fmt.Errorf("config: namespace is required")
	}
	if cfg.Image == "" {
		return nil, fmt.Errorf("config: image is required")
	}
	if cfg.Cluster == "" {
		return nil, fmt.Errorf("config: cluster is required")
	}
	if cfg.TestPackages == "" {
		return nil, fmt.Errorf("config: testPackages is required")
	}
	if cfg.GCS.Bucket == "" {
		return nil, fmt.Errorf("config: gcs.bucket is required")
	}
	if cfg.GCS.CredentialsSecret == "" {
		return nil, fmt.Errorf("config: gcs.credentialsSecret is required")
	}
	if cfg.UtilityImages.CloneRefs == "" {
		return nil, fmt.Errorf("config: utilityImages.cloneRefs is required")
	}
	if cfg.UtilityImages.InitUpload == "" {
		return nil, fmt.Errorf("config: utilityImages.initUpload is required")
	}
	if cfg.UtilityImages.Entrypoint == "" {
		return nil, fmt.Errorf("config: utilityImages.entrypoint is required")
	}
	if cfg.UtilityImages.Sidecar == "" {
		return nil, fmt.Errorf("config: utilityImages.sidecar is required")
	}

	if cfg.TimeoutMinutes == 0 {
		cfg.TimeoutMinutes = 120
	}
	if cfg.GracePeriodSeconds == 0 {
		cfg.GracePeriodSeconds = 15
	}

	return cfg, nil
}

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
	jobConfig     *JobConfig
	dryrun        bool
}

// NewGitHubEventsHandler creates and returns a new GitHubEventsHandler.
func NewGitHubEventsHandler(
	logger *logrus.Logger,
	prowJobClient prowv1.ProwJobInterface,
	githubClient githubClient,
	jobConfig *JobConfig,
	dryrun bool,
) *GitHubEventsHandler {
	return &GitHubEventsHandler{
		logger:        logger,
		prowJobClient: prowJobClient,
		githubClient:  githubClient,
		jobConfig:     jobConfig,
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
	cfg := h.jobConfig

	envKeys := make([]string, 0, len(cfg.Env))
	for k := range cfg.Env {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)
	envVars := make([]corev1.EnvVar, 0, len(cfg.Env))
	for _, k := range envKeys {
		envVars = append(envVars, corev1.EnvVar{Name: k, Value: cfg.Env[k]})
	}

	pathStrategy := prowapi.PathStrategyExplicit
	if cfg.GCS.PathStrategy != "" {
		pathStrategy = cfg.GCS.PathStrategy
	}

	presubmit := config.Presubmit{
		JobBase: config.JobBase{
			Name:    "coverage-auto",
			Agent:   string(prowapi.KubernetesAgent),
			Cluster: cfg.Cluster,
			Labels: map[string]string{
				"coverage-plugin": "true",
			},
			Spec: &corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Image: cfg.Image,
						Command: []string{
							"/usr/local/bin/entrypoint.sh",
							"/bin/sh",
							"-ce",
						},
						Args: []string{
							fmt.Sprintf("go test %s -coverprofile=${ARTIFACTS}/filtered.cov && covreport -i ${ARTIFACTS}/filtered.cov -o ${ARTIFACTS}/filtered.html", cfg.TestPackages),
						},
						Env:  envVars,
					},
				},
			},
			Namespace: &cfg.Namespace,
			UtilityConfig: config.UtilityConfig{
				Decorate: &decorate,
				DecorationConfig: &prowapi.DecorationConfig{
					Timeout:     &prowapi.Duration{Duration: time.Duration(cfg.TimeoutMinutes) * time.Minute},
					GracePeriod: &prowapi.Duration{Duration: time.Duration(cfg.GracePeriodSeconds) * time.Second},
					UtilityImages: &prowapi.UtilityImages{
						CloneRefs:  cfg.UtilityImages.CloneRefs,
						InitUpload: cfg.UtilityImages.InitUpload,
						Entrypoint: cfg.UtilityImages.Entrypoint,
						Sidecar:    cfg.UtilityImages.Sidecar,
					},
					GCSConfiguration: &prowapi.GCSConfiguration{
						Bucket:       cfg.GCS.Bucket,
						PathStrategy: pathStrategy,
					},
					GCSCredentialsSecret: pStr(cfg.GCS.CredentialsSecret),
				},
			},
		},
		Reporter: config.Reporter{
			Context:    "coverage-auto",
			SkipReport: true,
		},
	}
	return pjutil.NewPresubmit(*pr, pr.Base.SHA, presubmit, eventGUID, nil)
}

func pStr(s string) *string {
	return &s
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

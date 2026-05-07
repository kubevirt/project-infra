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

type Config struct {
	Defaults JobConfig            `yaml:"defaults"`
	Repos    map[string]JobConfig `yaml:"repos"`
}

func (c *Config) RepoConfig(repo string) (*JobConfig, bool) {
	repoCfg, ok := c.Repos[repo]
	if !ok {
		return nil, false
	}
	merged := c.Defaults
	if repoCfg.TestPackages != "" {
		merged.TestPackages = repoCfg.TestPackages
	}
	if repoCfg.Image != "" {
		merged.Image = repoCfg.Image
	}
	if repoCfg.Namespace != "" {
		merged.Namespace = repoCfg.Namespace
	}
	if repoCfg.Cluster != "" {
		merged.Cluster = repoCfg.Cluster
	}
	if len(repoCfg.Env) > 0 {
		merged.Env = repoCfg.Env
	}
	if repoCfg.TimeoutMinutes != 0 {
		merged.TimeoutMinutes = repoCfg.TimeoutMinutes
	}
	if repoCfg.GracePeriodSeconds != 0 {
		merged.GracePeriodSeconds = repoCfg.GracePeriodSeconds
	}
	if repoCfg.UtilityImages != (UtilityImagesConfig{}) {
		merged.UtilityImages = repoCfg.UtilityImages
	}
	if repoCfg.GCS != (GCSConfig{}) {
		merged.GCS = repoCfg.GCS
	}
	return &merged, true
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	cfg := &Config{}
	
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	d := cfg.Defaults
	if d.Namespace == "" {
		return nil, fmt.Errorf("config: defaults.namespace is required")
	}
	if d.Image == "" {
		return nil, fmt.Errorf("config: defaults.image is required")
	}
	if d.Cluster == "" {
		return nil, fmt.Errorf("config: defaults.cluster is required")
	}
	if d.GCS.Bucket == "" {
		return nil, fmt.Errorf("config: defaults.gcs.bucket is required")
	}
	if d.GCS.CredentialsSecret == "" {
		return nil, fmt.Errorf("config: defaults.gcs.credentialsSecret is required")
	}
	if d.UtilityImages.CloneRefs == "" {
		return nil, fmt.Errorf("config: defaults.utilityImages.cloneRefs is required")
	}
	if d.UtilityImages.InitUpload == "" {
		return nil, fmt.Errorf("config: defaults.utilityImages.initUpload is required")
	}
	if d.UtilityImages.Entrypoint == "" {
		return nil, fmt.Errorf("config: defaults.utilityImages.entrypoint is required")
	}
	if d.UtilityImages.Sidecar == "" {
		return nil, fmt.Errorf("config: defaults.utilityImages.sidecar is required")
	}

	if cfg.Defaults.TimeoutMinutes < 0 {
		return nil, fmt.Errorf("config: defaults.timeoutMinutes must not be negative")
	}
	if cfg.Defaults.GracePeriodSeconds < 0 {
		return nil, fmt.Errorf("config: defaults.gracePeriodSeconds must not be negative")
	}
	if cfg.Defaults.TimeoutMinutes == 0 {
		cfg.Defaults.TimeoutMinutes = 120
	}
	if cfg.Defaults.GracePeriodSeconds == 0 {
		cfg.Defaults.GracePeriodSeconds = 15
	}

	if len(cfg.Repos) == 0 {
		return nil, fmt.Errorf("config: at least one repo must be configured")
	}
	for repo, repoCfg := range cfg.Repos {
		if repoCfg.TestPackages == "" {
			return nil, fmt.Errorf("config: repos.%s.testPackages is required", repo)
		}
		if repoCfg.TimeoutMinutes < 0 {
			return nil, fmt.Errorf("config: repos.%s.timeoutMinutes must not be negative", repo)
		}
		if repoCfg.GracePeriodSeconds < 0 {
			return nil, fmt.Errorf("config: repos.%s.gracePeriodSeconds must not be negative", repo)
		}
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
	config        *Config
	dryrun        bool
}

// NewGitHubEventsHandler creates and returns a new GitHubEventsHandler.
func NewGitHubEventsHandler(
	logger *logrus.Logger,
	prowJobClient prowv1.ProwJobInterface,
	githubClient githubClient,
	config *Config,
	dryrun bool,
) *GitHubEventsHandler {
	return &GitHubEventsHandler{
		logger:        logger,
		prowJobClient: prowJobClient,
		githubClient:  githubClient,
		config:        config,
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
	pr *github.PullRequest, eventGUID string, cfg *JobConfig) prowapi.ProwJob {
	decorate := true

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

	pr := &prEvent.PullRequest
	repoKey := fmt.Sprintf("%s/%s", pr.Base.Repo.Owner.Login, pr.Base.Repo.Name)
	jobCfg, ok := h.config.RepoConfig(repoKey)
	if !ok {
		log.Infof("No coverage config for %s, skipping", repoKey)
		return
	}

	changes, err := h.getPullRequestChanges(pr)
	if err != nil {
		log.WithError(err).Error("Failed to get pull request changes")
		return
	}

	if !detectGoFileChanges(extractFilenames(changes)) {
		log.Info("No Go file changes detected, skipping coverage job")
		return
	}

	eventGUID := log.Data["event-guid"].(string)
	job := h.generateCoverageJob(pr, eventGUID, jobCfg)

	if h.dryrun {
		log.Infof("Dry-run: would create coverage job for PR #%d", pr.Number)
		return
	}

	if _, err := h.prowJobClient.Create(context.Background(), &job, metav1.CreateOptions{}); err != nil {
		log.WithError(err).Error("Failed to create coverage ProwJob")
		return
	}

	log.Infof("Created coverage job for PR #%d", pr.Number)
}

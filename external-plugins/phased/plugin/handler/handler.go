package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"

	pi_github "kubevirt.io/project-infra/pkg/github"
	kubeVirtLabels "kubevirt.io/project-infra/pkg/github/labels"

	"github.com/sirupsen/logrus"
	"sigs.k8s.io/prow/pkg/config"
	gitv2 "sigs.k8s.io/prow/pkg/git/v2"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/labels"
	"sigs.k8s.io/prow/pkg/pjutil"
)

var log *logrus.Logger

const Intro = "Required labels detected, running phase 2 presubmits:\n"

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
	CreateComment(org, repo string, number int, comment string) error
	GetPullRequestChanges(org, repo string, number int) ([]github.PullRequestChange, error)
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
}

type loadConfigBytesFunc func(h *GitHubEventsHandler, org, repo string) ([]byte, []byte, error)

var LoadConfigBytesFunc loadConfigBytesFunc = loadConfigBytes

type GitHubEventsHandler struct {
	eventsChan       <-chan *GitHubEvent
	logger           *logrus.Logger
	ghClient         githubClientInterface
	gitClientFactory gitv2.ClientFactory
	prowConfigPath   string
	jobsConfigBase   string
	prowLocation     string
}

func NewGitHubEventsHandler(
	eventsChan <-chan *GitHubEvent,
	logger *logrus.Logger,
	ghClient githubClientInterface,
	prowConfigPath string,
	jobsConfigBase string,
	prowLocation string,
	gitClientFactory gitv2.ClientFactory) *GitHubEventsHandler {

	return &GitHubEventsHandler{
		eventsChan:       eventsChan,
		logger:           logger,
		ghClient:         ghClient,
		prowConfigPath:   prowConfigPath,
		jobsConfigBase:   jobsConfigBase,
		prowLocation:     prowLocation,
		gitClientFactory: gitClientFactory,
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
		log.Infoln("Dropping irrelevant:", incomingEvent.Type, incomingEvent.GUID)
	}
}

// For unit tests, as we create a local git NewFakeClient
func (h *GitHubEventsHandler) SetLocalConfLoad() {
	LoadConfigBytesFunc = loadLocalConfigBytes
}

func (h *GitHubEventsHandler) handlePullRequestEvent(log *logrus.Entry, event *github.PullRequestEvent) {
	log.Infof("Handling updated pull request: %s [%d]", event.Repo.FullName, event.PullRequest.Number)

	if !h.shouldActOnPREvent(event) {
		return
	}

	org, repo, err := pi_github.OrgRepo(event.Repo.FullName)
	if err != nil {
		log.WithError(err).Errorf("Could not get org/repo from the event")
		return
	}

	shouldRun, err := h.shouldRunPhase2(org, repo, event.Label.Name, event.PullRequest.Number)
	if err != nil || !shouldRun {
		return
	}

	pr, err := h.ghClient.GetPullRequest(org, repo, event.PullRequest.Number)
	if err != nil {
		log.WithError(err).Errorf("Could not get PR number %d", event.PullRequest.Number)
		return
	}

	presubmits, err := h.loadPresubmits(*pr)
	if err != nil {
		log.WithError(err).Errorf("loadPresubmits failed")
		return
	}

	if presubmits == nil {
		return
	}

	toTest, err := listRequiredManual(h.ghClient, *pr, presubmits)
	if err != nil {
		log.WithError(err).Errorf("listRequiredManual failed")
		return
	}

	err = testRequested(h.ghClient, *pr, toTest)
	if err != nil {
		log.WithError(err).Errorf("testRequested failed")
		return
	}
}

func (h *GitHubEventsHandler) loadPresubmits(pr github.PullRequest) ([]config.Presubmit, error) {
	if pr.Base.Ref != "main" && pr.Base.Ref != "master" {
		return nil, nil
	}

	pc, err := h.loadProwConfig(pr.Base.Repo.FullName)
	if err != nil {
		log.WithError(err).Errorf("Could not load prow config")
		return nil, err
	}

	org, repo, err := pi_github.OrgRepo(pr.Base.Repo.FullName)
	if err != nil {
		log.WithError(err).Errorf("Could not parse repo name: %s", pr.Base.Repo.FullName)
		return nil, err
	}

	orgRepo := org + "/" + repo
	var presubmits []config.Presubmit
	for index, jobs := range pc.PresubmitsStatic {
		if index != orgRepo {
			continue
		}
		presubmits = append(presubmits, jobs...)
	}

	return presubmits, nil
}

func generateJobConfigURL(org, repo, prowLocation, jobsConfigBase string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s-presubmits.yaml",
		prowLocation, jobsConfigBase, org, repo, repo)
}

func (h *GitHubEventsHandler) shouldActOnPREvent(event *github.PullRequestEvent) bool {
	return event.Action == github.PullRequestActionLabeled
}

func (h *GitHubEventsHandler) shouldRunPhase2(org, repo, eventLabel string, prNum int) (bool, error) {
	l, err := h.ghClient.GetIssueLabels(org, repo, prNum)
	if err != nil {
		log.WithError(err).Errorf("Could not get PR labels")
		return false, err
	}

	return (eventLabel == labels.LGTM && github.HasLabel(labels.Approved, l)) ||
		(eventLabel == labels.Approved && github.HasLabel(labels.LGTM, l)) ||
		(eventLabel == kubeVirtLabels.SkipReview), nil
}

func catFile(log *logrus.Logger, gitDir, file, refspec string) ([]byte, int) {
	cmd := exec.Command("git", "-C", gitDir, "cat-file", "-p", fmt.Sprintf("%s:%s", refspec, file))
	log.Debugf("Executing git command: %+v", cmd.Args)
	out, _ := cmd.CombinedOutput()
	return out, cmd.ProcessState.ExitCode()
}

func writeTempFile(log *logrus.Logger, basedir string, content []byte) (string, error) {
	tmpfile, err := os.CreateTemp(basedir, "job-config")
	if err != nil {
		log.WithError(err).Errorf("Could not create temp file for job config.")
		return "", err
	}
	defer tmpfile.Close()
	_, err = tmpfile.Write(content)
	if err != nil {
		log.WithError(err).Errorf("Could not write data to file: %s", tmpfile.Name())
		return "", err
	}
	tmpfile.Sync()
	return tmpfile.Name(), nil
}

func fetchRemoteFile(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP request failed with status: %v", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func listRequiredManual(ghClient githubClientInterface, pr github.PullRequest, presubmits []config.Presubmit) ([]config.Presubmit, error) {
	if pr.Draft || pr.Merged || pr.State != "open" {
		return nil, nil
	}

	if pr.Mergable != nil && !*pr.Mergable {
		return nil, nil
	}

	org, repo, number, branch := pr.Base.Repo.Owner.Login, pr.Base.Repo.Name, pr.Number, pr.Base.Ref
	changes := config.NewGitHubDeferredChangedFilesProvider(ghClient, org, repo, number)
	toTest, err := pjutil.FilterPresubmits(defaultManualRequiredFilter, changes, branch, presubmits, log)
	if err != nil {
		return nil, err
	}

	return toTest, nil
}

var defaultManualRequiredFilter = manualRequiredFilter{}

type manualRequiredFilter struct{}

func (f manualRequiredFilter) ShouldRun(p config.Presubmit) (shouldRun bool, forcedToRun bool, defaultBehavior bool) {
	cond := !p.Optional && !p.AlwaysRun && p.RegexpChangeMatcher.RunIfChanged == "" &&
		p.RegexpChangeMatcher.SkipIfOnlyChanged == ""
	return cond, cond, false
}

func (f manualRequiredFilter) Name() string { return "manualRequiredFilter" }

func testRequested(ghClient githubClientInterface, pr github.PullRequest, requestedJobs []config.Presubmit) error {
	org, repo, err := pi_github.OrgRepo(pr.Base.Repo.FullName)
	if err != nil {
		log.WithError(err).Errorf("Could not parse repo name: %s", pr.Base.Repo.FullName)
		return err
	}

	// Until we update k8s.io/test-infra which allows to read require_manually_triggered_jobs policy
	if !(org == "kubevirt" && repo == "kubevirt") {
		return nil
	}

	var result string
	for _, job := range requestedJobs {
		result += "/test " + job.Name + "\n"
		log.Debugf("Found presubmit %s", job.Name)
	}

	if result != "" {
		result = Intro + result
		if err := ghClient.CreateComment(org, repo, pr.Number, result); err != nil {
			log.WithError(err).Errorf("CreateComment failed PR %d", pr.Number)
			return err
		}
	}

	return nil
}

func loadLocalConfigBytes(h *GitHubEventsHandler, org, repo string) ([]byte, []byte, error) {
	git, err := h.gitClientFactory.ClientFor(org, repo)
	if err != nil {
		log.WithError(err).Errorf("Could not get client for git")
		return nil, nil, err
	}

	prowConfigBytes, ret := catFile(log, git.Directory(), h.prowConfigPath, "HEAD")
	if ret != 0 {
		log.WithError(err).Errorf("Could not load Prow config %s", h.prowConfigPath)
		return nil, nil, err
	}

	jobConfigBytes, ret := catFile(log, git.Directory(), h.jobsConfigBase, "HEAD")
	if ret != 0 {
		log.WithError(err).Errorf("Could not load Prow config %s", h.jobsConfigBase)
		return nil, nil, err
	}

	return prowConfigBytes, jobConfigBytes, nil
}

func loadConfigBytes(h *GitHubEventsHandler, org, repo string) ([]byte, []byte, error) {
	prowConfigUrl := h.prowLocation + "/" + h.prowConfigPath
	prowConfigBytes, err := fetchRemoteFile(prowConfigUrl)
	if err != nil {
		log.WithError(err).Errorf("Could not fetch prow config from %s", prowConfigUrl)
		return nil, nil, err
	}

	jobConfigUrl := generateJobConfigURL(org, repo, h.prowLocation, h.jobsConfigBase)
	jobConfigBytes, err := fetchRemoteFile(jobConfigUrl)
	if err != nil {
		log.WithError(err).Errorf("Could not fetch prow config from %s", jobConfigUrl)
		return nil, nil, err
	}

	return prowConfigBytes, jobConfigBytes, nil
}

func (h *GitHubEventsHandler) loadProwConfig(prFullName string) (*config.Config, error) {
	tmpdir, err := os.MkdirTemp("", "prow-configs")
	if err != nil {
		log.WithError(err).Error("Could not create a temp directory to store configs.")
		return nil, err
	}
	defer os.RemoveAll(tmpdir)

	org, repo, err := pi_github.OrgRepo(prFullName)
	if err != nil {
		log.WithError(err).Errorf("Could not parse repo name: %s", prFullName)
		return nil, err
	}

	prowConfigBytes, jobConfigBytes, err := LoadConfigBytesFunc(h, org, repo)
	if err != nil {
		log.WithError(err).Errorf("Could not load prow config")
		return nil, err
	}

	prowConfigTmp, err := writeTempFile(log, tmpdir, prowConfigBytes)
	if err != nil {
		log.WithError(err).Errorf("Could not write temporary Prow config.")
		return nil, err
	}

	jobConfigTmp, err := writeTempFile(log, tmpdir, jobConfigBytes)
	if err != nil {
		log.WithError(err).Errorf("Could not write temporary Job config file")
		return nil, err
	}

	pc, err := config.Load(prowConfigTmp, jobConfigTmp, nil, "")
	if err != nil {
		log.WithError(err).Errorf("Could not load prow config")
		return nil, err
	}

	return pc, nil
}

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	k8s_v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/test-infra/prow/client/clientset/versioned/typed/prowjobs/v1"
	"k8s.io/test-infra/prow/config"
	gitv2 "k8s.io/test-infra/prow/git/v2"
	"k8s.io/test-infra/prow/github"
	"k8s.io/test-infra/prow/pjutil"
)

var log *logrus.Logger

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
	IsMember(string, string) (bool, error)
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
	prowClient       v1.ProwJobInterface
	ghClient         githubClientInterface
	gitClientFactory gitv2.ClientFactory
	prowConfigPath   string
	jobsConfigBase   string
	prowLocation     string
}

func NewGitHubEventsHandler(
	eventsChan <-chan *GitHubEvent,
	logger *logrus.Logger,
	prowClient v1.ProwJobInterface,
	ghClient githubClientInterface,
	prowConfigPath string,
	jobsConfigBase string,
	prowLocation string,
	gitClientFactory gitv2.ClientFactory) *GitHubEventsHandler {

	return &GitHubEventsHandler{
		eventsChan:       eventsChan,
		logger:           logger,
		prowClient:       prowClient,
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
	case "issue_comment":
		logrus.Infoln("Handling issue comment event")
		var event github.IssueCommentEvent
		if err := json.Unmarshal(incomingEvent.Payload, &event); err != nil {
			log.WithError(err).Error("Could not unmarshal event.")
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

func (h *GitHubEventsHandler) handlePullRequestEvent(log *logrus.Entry, event *github.IssueCommentEvent) {
	if !h.shouldActOnIssueComment(event) {
		return
	}

	org, repo, err := gitv2.OrgRepo(event.Repo.FullName)
	if err != nil {
		log.WithError(err).Errorf("Could not get org/repo from the event")
		return
	}

	if !(org == "kubevirt" && repo == "kubevirt") {
		return
	}

	pr, err := h.ghClient.GetPullRequest(org, repo, event.Issue.Number)
	if err != nil {
		log.WithError(err).Errorf("Could not get PR number %d", event.Issue.Number)
		return
	}

	if !isPRReady(*pr) {
		return
	}

	if !h.canUserTrigger(org, event.Comment.User.Login) {
		return
	}

	specifyCommentRe := regexp.MustCompile(`^/specify (\S+) (.+)$`)
	if !specifyCommentRe.MatchString(event.Comment.Body) {
		log.Errorf("Comment does not match the expected syntax.")
		return
	}

	matches := specifyCommentRe.FindStringSubmatch(event.Comment.Body)
	jobName := matches[1]
	labels := matches[2]

	presumbits, err := h.loadPresubmits(*pr)
	if err != nil {
		log.WithError(err).Errorf("loadPresubmits failed")
		return
	}

	if presumbits == nil {
		return
	}

	var presubmit config.Presubmit
	for _, p := range presumbits {
		if p.Name != jobName {
			continue
		}
		presubmit = p
		break
	}

	if presubmit.Name == "" {
		return
	}

	// TODO label need to be "()" ...
	presubmit.Spec.Containers[0].Env = append(presubmit.Spec.Containers[0].Env,
		k8s_v1.EnvVar{Name: "KUBEVIRT_LABEL_FILTER", Value: labels})

	//TODO
	//fmt.Printf("%+v\n", presubmit)

	job := pjutil.NewPresubmit(*pr, pr.Base.SHA, presubmit, event.GUID, map[string]string{})
	if job.Labels == nil {
		job.Labels = make(map[string]string)
	}
	job.Labels["prow-specify"] = "true"
	specifyName := strings.Join([]string{"specify", job.Spec.Job}, "-")
	job.Spec.Job = specifyName
	job.Spec.Context = specifyName
	_, err = h.prowClient.Create(context.Background(), &job, metav1.CreateOptions{})
	if err != nil {
		log.WithError(err).Errorf("Error creating prow job")
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

	org, repo, err := gitv2.OrgRepo(pr.Base.Repo.FullName)
	if err != nil {
		log.WithError(err).Errorf("Could not parse repo name: %s", pr.Base.Repo.FullName)
		return nil, err
	}

	orgRepo := org + "/" + repo
	var presumbits []config.Presubmit
	for index, jobs := range pc.PresubmitsStatic {
		if index != orgRepo {
			continue
		}
		presumbits = append(presumbits, jobs...)
	}

	return presumbits, nil
}

func generateJobConfigURL(org, repo, prowLocation, jobsConfigBase string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s-presubmits.yaml",
		prowLocation, jobsConfigBase, org, repo, repo)
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

func isPRReady(pr github.PullRequest) bool {
	if pr.Merged || pr.State != "open" {
		return false
	}

	if pr.Mergable != nil && !*pr.Mergable {
		return false
	}

	return true
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

	org, repo, err := gitv2.OrgRepo(prFullName)
	if err != nil {
		log.WithError(err).Errorf("Could not parse repo name: %s", prFullName)
		return nil, err
	}

	prowConfigBytes, jobConfigBytes, err := LoadConfigBytesFunc(h, org, repo)
	if err != nil {
		log.WithError(err).Errorf("Could not load prow config files")
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

func (h *GitHubEventsHandler) shouldActOnIssueComment(event *github.IssueCommentEvent) bool {
	return event.Issue.IsPullRequest() && event.Issue.State == "open" && event.Action == github.IssueCommentActionCreated
}

func (h *GitHubEventsHandler) canUserTrigger(org string, userName string) bool {
	isMember, err := h.ghClient.IsMember(org, userName)
	if err != nil {
		log.WithError(err).Errorln("Could not validate PR author with the repo org")
		return false
	}
	return isMember
}

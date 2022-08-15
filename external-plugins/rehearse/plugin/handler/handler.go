package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	prowapi "k8s.io/test-infra/prow/apis/prowjobs/v1"
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

// GitHubEvent represents a valid GitHub event in the events channel
type GitHubEvent struct {
	Type    string
	GUID    string
	Payload []byte
}

type GitHubEventsHandler struct {
	eventsChan       <-chan *GitHubEvent
	logger           *logrus.Logger
	prowClient       v1.ProwJobInterface
	ghClient         githubClientInterface
	gitClientFactory gitv2.ClientFactory
	prowConfigPath   string
	jobsConfigBase   string
	alwaysRun        bool
}

// NewGitHubEventsHandler returns a new github events handler
func NewGitHubEventsHandler(
	eventsChan <-chan *GitHubEvent,
	logger *logrus.Logger,
	prowClient v1.ProwJobInterface,
	ghClient githubClientInterface,
	prowConfigPath string,
	jobsConfigBase string,
	alwaysRun bool,
	gitClientFactory gitv2.ClientFactory) *GitHubEventsHandler {

	return &GitHubEventsHandler{
		eventsChan:       eventsChan,
		logger:           logger,
		prowClient:       prowClient,
		ghClient:         ghClient,
		prowConfigPath:   prowConfigPath,
		jobsConfigBase:   jobsConfigBase,
		alwaysRun:        alwaysRun,
		gitClientFactory: gitClientFactory,
	}
}

type githubClientInterface interface {
	IsMember(string, string) (bool, error)
	GetPullRequest(string, string, int) (*github.PullRequest, error)
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
		h.handlePullRequestUpdateEvent(eventLog, &event)
	case "issue_comment":
		logrus.Infoln("Handling issue comment event")
		var event github.IssueCommentEvent
		if err := json.Unmarshal(incomingEvent.Payload, &event); err != nil {
			log.WithError(err).Error("Could not unmarshal event.")
			return
		}
		h.handleIssueComment(eventLog, &event)
	default:
		log.Infoln("Dropping irrelevant:", incomingEvent.Type, incomingEvent.GUID)
	}
}

func (h *GitHubEventsHandler) handleIssueComment(log *logrus.Entry, event *github.IssueCommentEvent) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Warnf("Recovered during handling of an issue comment event: %s", event.GUID)
		}
	}()

	if !h.shouldActOnIssueComment(event) {
		log.Infoln("Skipping event - we shouldn't act on it")
		return
	}

	rehearseCommentRe := regexp.MustCompile(`(?m)^/rehearse$`)
	if !rehearseCommentRe.MatchString(event.Comment.Body) {
		return
	}

	if !h.validateEventUser(event.Repo.FullName, event.Comment.User.Login, event.Issue.Number) {
		log.Infoln("Skipping event - user validation failed")
		return
	}

	org, repo, err := gitv2.OrgRepo(event.Repo.FullName)
	if err != nil {
		log.WithError(err).Errorln("Could not get org and repo from comment")
	}
	pr, err := h.ghClient.GetPullRequest(org, repo, event.Issue.Number)
	if err != nil {
		log.WithError(err).Errorln("Could not get pull request for comment")
	}

	h.handleRehearsalForPR(log, pr, event.GUID)
}

func (h *GitHubEventsHandler) shouldActOnPREvent(event *github.PullRequestEvent) bool {
	switch event.Action {
	case github.PullRequestActionOpened, github.PullRequestActionSynchronize:
		return true
	default:
		return false
	}
}

func (h *GitHubEventsHandler) shouldActOnIssueComment(event *github.IssueCommentEvent) bool {
	if event.Issue.IsPullRequest() && event.Issue.State == "open" && event.Action == github.IssueCommentActionCreated {
		log.Infof("Event is PR: %t, event issue state: %s, event action: %s", event.Issue.IsPullRequest(), event.Issue.State, event.Action)
		return true
	}
	return false
}

func (h *GitHubEventsHandler) validateEventUser(repoFullName, userName string, prNumber int) bool {
	org, repo, err := gitv2.OrgRepo(repoFullName)
	if err != nil {
		log.WithError(err).Errorf("Could not get org/repo from the event")
	}
	pr, err := h.ghClient.GetPullRequest(org, repo, prNumber)
	if err != nil {
		log.WithError(err).Errorf("Could not get PR number %d", prNumber)
	}
	isMember, err := h.ghClient.IsMember(org, userName)
	if err != nil {
		log.WithError(err).Errorln("Could not validate PR author with the repo org")
		return false
	}
	return isMember || github.HasLabel("ok-to-test", pr.Labels)
}

func (h *GitHubEventsHandler) handlePullRequestUpdateEvent(log *logrus.Entry, event *github.PullRequestEvent) {
	defer func() {
		if r := recover(); r != nil {
			h.logger.Warnf("Recovered during handling of a pull request event: %s", event.GUID)
		}
	}()

	if !h.alwaysRun {
		return
	}

	log.Infof("Handling updated pull request: %s [%d]", event.Repo.FullName, event.PullRequest.Number)

	if !h.shouldActOnPREvent(event) {
		log.Infoln("Skipping event. Not of our interest.")
		return
	}

	if !h.validateEventUser(event.Repo.FullName, event.Sender.Login, event.PullRequest.Number) {
		log.Infoln("Skipping event. User is not authorized.")
		return
	}

	h.handleRehearsalForPR(log, &event.PullRequest, event.GUID)
}

func (h *GitHubEventsHandler) handleRehearsalForPR(log *logrus.Entry, pr *github.PullRequest, eventGUID string) {
	repo, org, err := gitv2.OrgRepo(pr.Head.Repo.FullName)
	if err != nil {
		log.WithError(err).Errorf("Could not parse repo name: %s", pr.Head.Repo.FullName)
		return
	}
	log.Infoln("Generating git client")
	git, err := h.gitClientFactory.ClientFor(repo, org)
	if err != nil {
		return
	}

	log.Infoln("Fetching target branch", pr.Base.Ref)
	err = git.FetchRef(pr.Base.Ref)
	if err != nil {
		log.WithError(err).Error("Could not fetch pull request's target branch.")
		return
	}
	log.Infoln("Fetching PR head ref", pr.Head.SHA)
	err = git.FetchRef(pr.Head.SHA)
	if err != nil {
		log.WithError(err).Error("Could not fetch pull request's head ref.")
		return
	}
	log.Infoln("Rebasing the PR on the target branch")
	git.Config("user.email", "kubevirtbot@redhat.com")
	git.Config("user.name", "Kubevirt Bot")
	err = git.MergeAndCheckout(pr.Base.Ref, string(github.MergeSquash), pr.Head.SHA)
	if err != nil {
		log.WithError(err).Error("Could not rebase the PR on the target branch.")
		return
	}
	log.Infoln("Getting diff")
	changedFiles, err := git.Diff("HEAD", pr.Base.Ref)
	if err != nil {
		log.WithError(err).Error("Could not calculate diff for PR.")
		return
	}
	log.Infoln("Changed files:", changedFiles)
	changedJobConfigs, err := h.modifiedJobConfigs(changedFiles)
	if err != nil {
		log.WithError(err).Error("Could not calculate absolute paths for modified job configs")
		return
	}
	log.Infoln("Changed job configs:", changedJobConfigs)
	headConfigs, err := h.loadConfigsAtRef(changedJobConfigs, git, pr.Head.SHA)
	if err != nil {
		log.WithError(err).Errorf(
			"Could not load job configs from head ref: %s", pr.Head.SHA)
	}

	baseConfigs, err := h.loadConfigsAtRef(changedJobConfigs, git, pr.Base.SHA)
	if err != nil {
		log.WithError(err).Errorf(
			"Could not load job configs from base ref: %s", pr.Base.SHA)
	}
	log.Infoln("Base configs:", baseConfigs)

	prowjobs := h.generateProwJobs(headConfigs, baseConfigs, pr, eventGUID)
	log.Infof("Will create %d jobs", len(prowjobs))
	for _, job := range prowjobs {
		if job.Labels == nil {
			job.Labels = make(map[string]string)
		}
		job.Labels["rehearsal"] = "true"
		job.Labels["rehearsal-for-pull-request"] = strconv.Itoa(pr.Number)
		rehearsalName := strings.Join([]string{"rehearsal", job.Spec.Job}, "-")
		job.Spec.Job = rehearsalName
		job.Spec.Context = rehearsalName
		_, err := h.prowClient.Create(context.Background(), &job, metav1.CreateOptions{})
		if err != nil {
			log.WithError(err).Errorf("Failed to create prow job: %s", job.Spec.Job)
		}
		log.Infof("Created a rehearse job: %s", job.Name)
	}
}

const rehearsalRestrictedAnnotation = "rehearsal.restricted"

func rehearsalRestricted(job prowapi.ProwJob) bool {
	annotations := job.GetAnnotations()
	if annotations == nil {
		return false
	}
	isRestricted, restrictedAnnotationExists := annotations[rehearsalRestrictedAnnotation]
	return restrictedAnnotationExists && isRestricted == "true"
}

func (h *GitHubEventsHandler) generateProwJobs(
	headConfigs, baseConfigs map[string]*config.Config, pr *github.PullRequest, eventGUID string) []prowapi.ProwJob {
	var jobs []prowapi.ProwJob

	for path, headConfig := range headConfigs {
		baseConfig, _ := baseConfigs[path]
		jobs = append(jobs, h.generatePresubmits(headConfig, baseConfig, pr, eventGUID)...)
	}

	return jobs
}

func (h *GitHubEventsHandler) generatePresubmits(
	headConfig, baseConfig *config.Config, pr *github.PullRequest, eventGUID string) []prowapi.ProwJob {
	var jobs []prowapi.ProwJob

	// We need to flatten the jobs because later on we need
	// to calculate the modified jobs and it will make the lookup
	// much more efficient.
	headPresubmits := hashPresubmitsConfig(headConfig.PresubmitsStatic)
	basePresubmits := hashPresubmitsConfig(baseConfig.PresubmitsStatic)

	for presubmitKey, headPresubmit := range headPresubmits {
		basePresubmit, exists := basePresubmits[presubmitKey]

		if exists && reflect.DeepEqual(basePresubmit.Spec, headPresubmit.Spec) {
			continue
		}
		log.Infof("Detected modified or new presubmit: %s.", headPresubmit.Name)

		job := pjutil.NewPresubmit(*pr, pr.Base.SHA, headPresubmit, eventGUID, map[string]string{})

		if rehearsalRestricted(job) {
			h.logger.Infof("Skipping rehersal job for: %s because it is restricted", job.Name)
			continue
		}

		repoOrg := repoFromJobKey(presubmitKey)
		org, repo, err := gitv2.OrgRepo(repoOrg)
		if err != nil {
			log.Errorf(
				"Could not extract repo and org from job key: %s. Job name: %s",
				presubmitKey, headPresubmit.Name)
		}

		headBranchName, err := discoverHeadBranchName(org, repo, headPresubmit.CloneURI)
		if err != nil {
			headBranchName = pr.Base.Ref
		}

		if repoOrg != pr.Base.Repo.FullName {
			job.Spec.ExtraRefs = append(job.Spec.ExtraRefs, makeTargetRepoRefs(job.Spec.ExtraRefs, org, repo, headBranchName))
		}
		jobs = append(jobs, job)
	}
	return jobs
}

func (h *GitHubEventsHandler) loadConfigsAtRef(
	changedJobConfigs []string, git gitv2.RepoClient, ref string) (map[string]*config.Config, error) {
	configs := map[string]*config.Config{}

	tmpdir, err := ioutil.TempDir("", "prow-configs")
	if err != nil {
		log.WithError(err).Error("Could not create a temp directory to store configs.")
		return nil, err
	}
	defer os.RemoveAll(tmpdir)
	// In order to actually support multi-threaded access to the git repo, we can't checkout any refs or assume that
	// the repo is checked out with any refspec. Instead, we use git cat-file to read the files that we need to the
	// memory and write them to a temp file. We need the temp file because the current version of Prow's config module
	// can't load the configs from the memory.
	prowConfigBytes, ret := catFile(log, git.Directory(), h.prowConfigPath, ref)
	if ret != 0 && ret != 128 {
		log.WithError(err).Errorf("Could not load Prow config [%s] at ref: %s", h.prowConfigPath, ref)
		return nil, err
	}
	log.Infoln("File:", string(prowConfigBytes))
	prowConfigTmp, err := writeTempFile(log, tmpdir, prowConfigBytes)
	if err != nil {
		log.WithError(err).Errorf("Could not write temporary Prow config.")
		return nil, err
	}

	for _, changedJobConfig := range changedJobConfigs {
		bytes, ret := catFile(log, git.Directory(), changedJobConfig, ref)
		if ret == 128 {
			// 128 means that the file was probably deleted in the PR or doesn't exists
			// so to avoid an edge case where we need to take care of a null pointer, we
			// generate an empty pc w/o jobs.
			configs[changedJobConfig] = &config.Config{}
			continue
		} else if ret != 0 {
			log.Errorf("Could not read job config from path %s at git ref: %s", changedJobConfig, ref)
			return nil, fmt.Errorf("could not read job config from git ref")
		}
		jobConfigTmp, err := writeTempFile(log, tmpdir, bytes)
		if err != nil {
			log.WithError(err).Infoln("Could not write temp file")
			return nil, err
		}
		pc, err := config.Load(prowConfigTmp, jobConfigTmp, nil, "")
		if err != nil {
			log.WithError(err).Errorf("Could not load job config from path %s at git ref %s", jobConfigTmp, ref)
			return nil, err
		}
		configs[changedJobConfig] = pc
	}

	return configs, nil
}

// modifiedJobConfigs generates an array of absolute paths for modified job configs
func (h *GitHubEventsHandler) modifiedJobConfigs(changedFiles []string) ([]string, error) {
	var absModifiedProwConfigs []string
	for _, changedFile := range changedFiles {
		if strings.HasPrefix(changedFile, h.jobsConfigBase) {
			if strings.HasSuffix(changedFile, ".yaml") || strings.HasSuffix(changedFile, ".yml") {
				log.Infof("A modified config found: %s", changedFile)
				absModifiedProwConfigs = append(absModifiedProwConfigs, changedFile)
			}
		}
		log.Debugf("Skipping file: %s. Not a Prow/Jobs config", changedFile)
	}
	return absModifiedProwConfigs, nil
}

func jobKeyFunc(repo string, presubmit config.JobBase) string {
	return fmt.Sprintf("%s#%s", repo, presubmit.Name)
}

func repoFromJobKey(jobKey string) string {
	s := strings.Split(jobKey, "#")
	r := s[:1]
	return strings.Join(r, "/")
}

func hashPeriodicsConfig(periodics []config.Periodic) map[string]config.Periodic {
	p := map[string]config.Periodic{}
	for _, periodic := range periodics {
		p[periodic.JobBase.Name] = periodic
	}
	return p
}

func hashPresubmitsConfig(presubmits map[string][]config.Presubmit) map[string]config.Presubmit {
	presubmitsFlat := map[string]config.Presubmit{}
	for repo, presubmitsForRepo := range presubmits {
		for _, presubmit := range presubmitsForRepo {
			presubmitsFlat[jobKeyFunc(repo, presubmit.JobBase)] = presubmit
		}
	}
	return presubmitsFlat
}

// catFile executes a git cat-file command in the specified git dir and returns bytes representation of the file
func catFile(log *logrus.Logger, gitDir, file, refspec string) ([]byte, int) {
	cmd := exec.Command("git", "-C", gitDir, "cat-file", "-p", fmt.Sprintf("%s:%s", refspec, file))
	log.Debugf("Executing git command: %+v", cmd.Args)
	out, _ := cmd.CombinedOutput()
	return out, cmd.ProcessState.ExitCode()
}

func writeTempFile(log *logrus.Logger, basedir string, content []byte) (string, error) {
	tmpfile, err := ioutil.TempFile(basedir, "job-config")
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

func makeTargetRepoRefs(refs []prowapi.Refs, org, repo, ref string) prowapi.Refs {
	return prowapi.Refs{
		Repo:    repo,
		Org:     org,
		WorkDir: !workdirAlreadyDefined(refs),
		BaseRef: ref,
	}
}

func workdirAlreadyDefined(refs []prowapi.Refs) bool {
	exists := false
	for _, ref := range refs {
		exists = exists || ref.WorkDir
	}
	return exists
}

func discoverHeadBranchName(org, repo, cloneURI string) (string, error) {
	sourceURL := fmt.Sprintf("https://github.com/%s/%s.git", org, repo)
	if cloneURI != "" {
		sourceURL = cloneURI
	}

	// Create the remote with repository URL
	rem := git.NewRemote(memory.NewStorage(), &gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{sourceURL},
	})

	// We can then use every Remote functions to retrieve wanted information
	refs, err := rem.List(&git.ListOptions{})
	if err != nil {
		return "", err
	}

	var headBranch string
	for _, ref := range refs {
		if ref.Type() == plumbing.SymbolicReference && ref.Name().String() == "HEAD" {
			headBranch = strings.Split(ref.Target().String(), "/")[2]
			break
		}
	}
	if headBranch == "" {
		headBranch = "master"
	}
	return headBranch, nil
}

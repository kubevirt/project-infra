package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	pi_github "kubevirt.io/project-infra/pkg/github"

	"github.com/r3labs/diff/v3"
	"sigs.k8s.io/prow/pkg/repoowners"

	"github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	prowapi "sigs.k8s.io/prow/pkg/apis/prowjobs/v1"
	v1 "sigs.k8s.io/prow/pkg/client/clientset/versioned/typed/prowjobs/v1"
	"sigs.k8s.io/prow/pkg/config"
	gitv2 "sigs.k8s.io/prow/pkg/git/v2"
	"sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/prow/pkg/pjutil"
)

const basicHelpCommentText = `
<details>
<summary>Further information on rehearsals</summary>

A rehearsal can be triggered for all jobs by commenting either ` + "`/rehearse`" + ` or ` + "`/rehearse all`" + ` on this PR.

A rehearsal for a specific job can be triggered by commenting ` + "`/rehearse {job-name}`" + `.

Commenting ` + "`/rehearse ?`" + ` triggers a comment with a list of jobs that can be rehearsed.

A pull request can be rehearsed if either the user is authorized to rehearse or the pull
request has the ` + "`ok-to-rehearse`" + ` label.

Authorized users are the group of users that are members of the KubeVirt GitHub 
organization AND either are approvers[1] for all files in the pull request or are
top-level approvers[1] in the ` + "`project-infra`" + ` project.

[1]: see [OWNERS](https://www.kubernetes.dev/docs/guide/owners/#owners) file definition for reference.
</details>
`

var log *logrus.Logger

// rehearseCommentRe matches either the sole command, i.e.
// /rehearse
// or the command followed by a job name which we then extract by the
// capturing group, i.e.
// /rehearse job-name
var rehearseCommentRe = regexp.MustCompile(`(?m)^/rehearse\s*?($|\s.*)`)

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
	ownersClient     repoOwnersClient
	prowConfigPath   string
	jobsConfigBase   string
	alwaysRun        bool
}

// NewGitHubEventsHandler returns a new github events handler
func NewGitHubEventsHandler(eventsChan <-chan *GitHubEvent, logger *logrus.Logger, prowClient v1.ProwJobInterface, ghClient githubClientInterface, prowConfigPath string, jobsConfigBase string, alwaysRun bool, gitClientFactory gitv2.ClientFactory, ownersClient repoOwnersClient) *GitHubEventsHandler {

	return &GitHubEventsHandler{
		eventsChan:       eventsChan,
		logger:           logger,
		prowClient:       prowClient,
		ghClient:         ghClient,
		prowConfigPath:   prowConfigPath,
		jobsConfigBase:   jobsConfigBase,
		alwaysRun:        alwaysRun,
		gitClientFactory: gitClientFactory,
		ownersClient:     ownersClient,
	}
}

type githubClientInterface interface {
	IsMember(string, string) (bool, error)
	GetPullRequest(string, string, int) (*github.PullRequest, error)
	CreateComment(org, repo string, number int, comment string) error
	GetIssueLabels(org, repo string, number int) ([]github.Label, error)
}

type repoOwnersClient interface {
	LoadRepoOwners(org, repo, base string) (repoowners.RepoOwner, error)
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

	if !rehearseCommentRe.MatchString(event.Comment.Body) {
		return
	}

	org, repo, err := pi_github.OrgRepo(event.Repo.FullName)
	if err != nil {
		log.WithError(err).Errorf("Could not get org/repo from the event")
	}
	pr, err := h.ghClient.GetPullRequest(org, repo, event.Issue.Number)
	if err != nil {
		log.WithError(err).Errorf("Could not get PR number %d", event.Issue.Number)
	}
	repoClient, err := h.getRebasedRepoClient(log, pr, org, repo)
	if err != nil {
		log.WithError(err).Error("could not get repo client")
		return
	}
	log.Infoln("Getting diff")
	changedFiles, err := repoClient.Diff(pr.Base.SHA, "HEAD")
	if err != nil {
		log.WithError(err).Error("Could not calculate diff for PR.")
		return
	}
	rehearse, message := h.canUserRehearse(org, repo, pr, event.Comment.User.Login, changedFiles)
	if !rehearse {
		if message != "" {
			err = h.ghClient.CreateComment(org, repo, pr.Number, message)
			if err != nil {
				log.WithError(err).Errorf("Failed to create comment on %s/%s PR: %d", org, repo, pr.Number)
			}
			return
		}
		log.Infoln("Skipping event - user validation failed")
		return
	}
	h.handleRehearsalForPR(log, pr, event.GUID, event.Comment.Body)
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

func (h *GitHubEventsHandler) canUserRehearse(org string, repo string, pr *github.PullRequest, userName string, changedFiles []string) (canUserRehearse bool, message string) {

	userNameLowerCase := strings.ToLower(userName)

	owners, err := h.ownersClient.LoadRepoOwners(org, repo, pr.Base.Ref)
	if err != nil {
		log.WithError(err).Errorln(fmt.Sprintf("Could not load owners on org %s and repo %s", org, repo))
		return false, ""
	}

	// if user is top level approver, then
	// rehearsal is allowed
	// even if the author is not an org member
	// top level approvers is only a very limited circle of people
	topLevelApprovers := owners.TopLevelApprovers()
	for topLevelApprover := range topLevelApprovers {
		if userNameLowerCase == strings.ToLower(topLevelApprover) {
			return true, ""
		}
	}

	// if author is not a member of the org, then
	// rehearsal is not allowed
	// in order to avoid compromising the org with harmful PRs
	isAuthorMember, err := h.ghClient.IsMember(org, pr.User.Login)
	if err != nil {
		log.WithError(err).Errorln("Could not validate PR author with the repo org")
		return false, ""
	}
	if !isAuthorMember {
		log.Warnln("Pr author is not a member of the repo org")
		return false, ""
	}

	// if the user that issues rehearsal is not a member of the org, then
	// rehearsal is not allowed
	// in order to avoid denial of service type attacks
	isUserMember, err := h.ghClient.IsMember(org, userNameLowerCase)
	if err != nil {
		log.WithError(err).Errorln("Could not validate user with the repo org")
		return false, ""
	}
	if !isUserMember {
		log.Warnln("User is not a member of the repo org")
		return false, ""
	}

	// check if ok-to-rehearse label is present
	issueLabels, err := h.ghClient.GetIssueLabels(org, repo, pr.Number)
	if err != nil {
		log.WithError(err).WithField("pull_request_url", fmt.Sprintf("github.com/%s/%s/pull/%d", org, repo, pr.Number)).Errorln("Could not get labels for pull request")
		return false, ""
	}
	for _, label := range issueLabels {
		if label.Name == OKToRehearse {
			return true, ""
		}
	}

	// if user is not a leaf approver for all the files then
	// rehearsal is not allowed
	// to ensure that scope of rehearsal is not extended beyond responsibilities
	for _, changedFile := range changedFiles {
		isLeafApprover := false
		var leafApprover string
		for leafApprover = range owners.LeafApprovers(changedFile) {
			if strings.ToLower(leafApprover) == userNameLowerCase {
				isLeafApprover = true
				break
			}
		}

		if !isLeafApprover {
			tlas := make([]string, 0, len(topLevelApprovers))
			for tla := range topLevelApprovers {
				tlas = append(tlas, tla)
			}
			return false, fmt.Sprintf(`⚠️ @%s you need to be an approver for all the files to run rehearsal.

@%s can help run the rehearsal.

<details>

If that doesn't work, ping someone from this list:
* %s
</details>
`, userName, leafApprover, strings.Join(tlas, "\n* "))
		}
	}
	return true, ""
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

	org, repo, err := pi_github.OrgRepo(event.Repo.FullName)
	if err != nil {
		log.WithError(err).Errorf("Could not get org/repo from the event")
	}
	pr, err := h.ghClient.GetPullRequest(org, repo, event.PullRequest.Number)
	if err != nil {
		log.WithError(err).Errorf("Could not get PR number %d", event.PullRequest.Number)
	}
	repoClient, err := h.getRebasedRepoClient(log, pr, org, repo)
	if err != nil {
		log.WithError(err).Error("could not get repo client")
		return
	}
	log.Infoln("Getting diff")
	changedFiles, err := repoClient.Diff(pr.Base.SHA, "HEAD")
	if err != nil {
		log.WithError(err).Error("Could not calculate diff for PR.")
		return
	}
	rehearse, message := h.canUserRehearse(org, repo, pr, event.Sender.Login, changedFiles)
	if !rehearse {
		if message != "" {
			err = h.ghClient.CreateComment(org, repo, pr.Number, message)
			if err != nil {
				log.WithError(err).Errorf("Failed to create comment on %s/%s PR: %d", org, repo, pr.Number)
			}
			return
		}
		log.Infoln("Skipping event. User is not authorized.")
		return
	}

	h.handleRehearsalForPR(log, &event.PullRequest, event.GUID, "")
}

func (h *GitHubEventsHandler) handleRehearsalForPR(log *logrus.Entry, pr *github.PullRequest, eventGUID string, commentBody string) {
	org, repo, err := pi_github.OrgRepo(pr.Base.Repo.FullName)
	if err != nil {
		log.WithError(err).Errorf("Could not parse repo name: %s", pr.Base.Repo.FullName)
		return
	}
	repoClient, err := h.getRebasedRepoClient(log, pr, org, repo)
	if err != nil {
		log.WithError(err).Error("could not get repo client")
		return
	}
	log.Infoln("Getting diff")
	changedFiles, err := repoClient.Diff(pr.Base.SHA, "HEAD")
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
	headConfigs, err := h.loadConfigsAtRef(changedJobConfigs, repoClient, "HEAD")
	if err != nil {
		log.WithError(err).Errorf(
			"Could not load job configs from head ref: %s", "HEAD")
	}

	baseConfigs, err := h.loadConfigsAtRef(changedJobConfigs, repoClient, pr.Base.SHA)
	if err != nil {
		log.WithError(err).Errorf(
			"Could not load job configs from base ref: %s", pr.Base.SHA)
	}
	log.Infoln("Base configs:", baseConfigs)

	prowjobs := h.generateProwJobs(headConfigs, baseConfigs, pr, eventGUID)
	jobNames := h.extractJobNamesFromComment(commentBody)
	if len(jobNames) == 1 && jobNames[0] == "?" {
		var prowJobNames []string
		for _, prowJob := range prowjobs {
			prowJobNames = append(prowJobNames, prowJob.Spec.Job)
		}
		commentText := fmt.Sprintf(`Rehearsal is available for the following jobs in this PR:

`+"```"+`
%s
`+"```"+`

`+basicHelpCommentText, strings.Join(prowJobNames, "\n"))
		err := h.ghClient.CreateComment(org, repo, pr.Number, commentText)
		if err != nil {
			log.WithError(err).Errorf("Failed to create comment on %s/%s PR: %d", org, repo, pr.Number)
		}
		return
	}

	prowjobs = h.filterProwJobsByJobNames(prowjobs, jobNames)

	log.Infof("Will create %d jobs", len(prowjobs))
	var rehearsalsGenerated []string
	var rehearsalsFailed []string
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
			rehearsalsFailed = append(rehearsalsFailed, rehearsalName)
			log.WithError(err).Errorf("Failed to create prow job: %s", job.Spec.Job)
			continue
		}
		rehearsalsGenerated = append(rehearsalsGenerated, rehearsalName)
		log.Infof("Created a rehearse job: %s", job.Name)
	}
	commentText := fmt.Sprintf(`Rehearsal jobs created for this PR:

`+"```"+`
%s
`+"```"+`

`+basicHelpCommentText, strings.Join(rehearsalsGenerated, "\n"))
	err = h.ghClient.CreateComment(org, repo, pr.Number, commentText)
	if err != nil {
		log.WithError(err).Errorf("Failed to create comment on %s/%s PR: %d", org, repo, pr.Number)
	}
	if len(rehearsalsFailed) > 0 {
		commentText = fmt.Sprintf(`Rehearsal jobs failed to create for this PR:

`+"```"+`
%s
`+"```"+`

`+basicHelpCommentText, strings.Join(rehearsalsFailed, "\n"))
		err = h.ghClient.CreateComment(org, repo, pr.Number, commentText)
		if err != nil {
			log.WithError(err).Errorf("Failed to create comment on %s/%s PR: %d", org, repo, pr.Number)
		}
	}
}

func (h *GitHubEventsHandler) getRebasedRepoClient(log *logrus.Entry, pr *github.PullRequest, org string, repo string) (gitv2.RepoClient, error) {
	log.Debugln("Generating git client")
	rebasedRepoClient, err := h.gitClientFactory.ClientFor(org, repo)
	if err != nil {
		return nil, fmt.Errorf("could not generate repo client: %w", err)
	}
	log.Debugln("Rebasing the PR on the target branch")
	err = rebasedRepoClient.Config("user.email", "kubevirtbot@redhat.com")
	if err != nil {
		return nil, fmt.Errorf("could not change repo config: %w", err)
	}
	err = rebasedRepoClient.Config("user.name", "Kubevirt Bot")
	if err != nil {
		return nil, fmt.Errorf("could not change repo config: %w", err)
	}
	err = rebasedRepoClient.MergeAndCheckout(pr.Base.SHA, "squash", pr.Head.SHA)
	if err != nil {
		return nil, fmt.Errorf("could not rebase the PR on the target branch: %w", err)
	}
	return rebasedRepoClient, nil
}

func (h *GitHubEventsHandler) extractJobNamesFromComment(body string) []string {
	if body == "" {
		return nil
	}
	var jobNames []string
	allStringSubmatch := rehearseCommentRe.FindAllStringSubmatch(body, -1)
	for _, subMatches := range allStringSubmatch {
		if len(subMatches) < 2 {
			continue
		}
		trimmedJobName := strings.TrimSpace(subMatches[1])
		if trimmedJobName == "" || trimmedJobName == "all" {
			continue
		}
		jobNames = append(jobNames, trimmedJobName)
	}
	return jobNames
}

func (h *GitHubEventsHandler) filterProwJobsByJobNames(prowjobs []prowapi.ProwJob, jobNames []string) []prowapi.ProwJob {
	if len(jobNames) == 0 {
		return prowjobs
	}
	jobNamesToFilter := map[string]struct{}{}
	for _, jobName := range jobNames {
		jobNamesToFilter[jobName] = struct{}{}
	}
	var filteredProwJobs []prowapi.ProwJob
	for _, prowjob := range prowjobs {
		if _, exists := jobNamesToFilter[prowjob.Spec.Job]; !exists {
			continue
		}
		filteredProwJobs = append(filteredProwJobs, prowjob)
	}
	return filteredProwJobs
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
		baseConfig, ok := baseConfigs[path]
		if !ok {
			log.Errorf("Path %s not found in base configs", path)
		}
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

		if exists && reflect.DeepEqual(basePresubmit, headPresubmit) {
			continue
		}
		log.Infof("Detected modified or new presubmit: %s.", headPresubmit.Name)
		changelog, err := diff.Diff(basePresubmit, headPresubmit)
		if err != nil {
			log.Errorf("could not diff presubmits: %v", err)
		}
		log.Infof("differences detected:/n%v", changelog)

		// respect the Branches configuration for the job, i.e. avoid always running against HEAD
		branches := headPresubmit.Branches
		if len(branches) == 0 {
			branches = []string{"HEAD"}
		}

		// since we can have multiple branches we need to create one job per branch
		for _, branch := range branches {
			job := pjutil.NewPresubmit(*pr, pr.Base.SHA, headPresubmit, eventGUID, map[string]string{})

			if rehearsalRestricted(job) {
				h.logger.Infof("Skipping rehersal job for: %s because it is restricted", job.Name)
				continue
			}

			repoOrg := repoFromJobKey(presubmitKey)
			org, repo, err := pi_github.OrgRepo(repoOrg)
			if err != nil {
				log.Errorf(
					"Could not extract repo and org from job key: %s. Job name: %s",
					presubmitKey, headPresubmit.Name)
			}

			var targetBranchName string
			if branch == "HEAD" {
				targetBranchName, err = discoverHeadBranchName(org, repo, headPresubmit.CloneURI)
				if err != nil {
					targetBranchName = pr.Base.Ref
				}
			} else {
				targetBranchName = branch
			}

			if repoOrg != pr.Base.Repo.FullName {
				job.Spec.ExtraRefs = append(job.Spec.ExtraRefs, makeTargetRepoRefs(job.Spec.ExtraRefs, org, repo, targetBranchName))
			}
			jobs = append(jobs, job)
		}
	}
	return jobs
}

func (h *GitHubEventsHandler) loadConfigsAtRef(
	changedJobConfigs []string, git gitv2.RepoClient, ref string) (map[string]*config.Config, error) {
	configs := map[string]*config.Config{}

	tmpdir, err := os.MkdirTemp("", "prow-configs")
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
		// `config.Load` sets field `.JobBase.SourcePath` inside each job to the path from where the config was
		// read, thus a deep equals can not succeed if two (otherwise identical) configs are read from different
		// directories as we do here
		// thus we need to reset the SourcePath to the original value for each job config
		for _, presubmits := range pc.PresubmitsStatic {
			for index := range presubmits {
				presubmits[index].JobBase.SourcePath = path.Join(git.Directory(), changedJobConfig)
			}
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

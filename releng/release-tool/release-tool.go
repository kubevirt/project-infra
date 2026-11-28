package main

import (
	"bytes"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	yamlv3 "gopkg.in/yaml.v3"
	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/yaml"

	"golang.org/x/oauth2"

	"github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v32/github"
)

type blockerListCacheEntry struct {
	allBlockerIssues []*github.Issue
	allBlockerPRs    []*github.PullRequest
}

type releaseData struct {
	repoDir   string
	infraDir  string
	cacheDir  string
	repoUrl   string
	infraUrl  string
	forkOwner string
	repo      string
	org       string
	newBranch string

	tagBranch        string
	tag              string
	promoteRC        string
	promoteRCTime    time.Time
	previousTag      string
	releaseNotesFile string
	skipReleaseNotes bool

	force bool

	gitUser         string
	gitEmail        string
	gitToken        string
	githubClient    *github.Client
	githubTokenPath string

	dryRun bool

	// github cached results
	allReleases      []*github.RepositoryRelease
	allBranches      []*github.Branch
	blockerListCache map[string]*blockerListCacheEntry

	now time.Time
}

// Allow mocking for tests
var gitCommand = _gitCommand

// rpmURLsPresubmitTemplate is the template of the rpm urls presubmit job
// item. It is appended to the job config of a release branch by the
// release-tool when the branch is forked (forkProwJobs); branches forked
// before the job existed get it backfilled via --add-rpm-urls-job.
//
// The job deliberately does not live in the main job config: it must not
// run on main, where rpms are updated before they are mirrored.
//
// The job runs hack/rpm-mirror-checker from the repository checkout: the
// checker is backported to the release branch together with the job.
//
// The template is rendered by the release-tool with the repository name,
// the release branch and the release version (see rpmURLsPresubmitData).
// The rendered item is a single yaml list entry so that it can be
// inserted into an existing job config without rewriting it.
//
//go:embed templates/rpm-urls-presubmit.yaml
var rpmURLsPresubmitTemplate string

// rpmURLsPresubmitJobTemplate is the parsed rpm urls presubmit job
// template, compiled once and reused for every rendering
var rpmURLsPresubmitJobTemplate = template.Must(template.New("rpm-urls-presubmit.yaml").Parse(rpmURLsPresubmitTemplate))

// rpmURLsPresubmitData holds the values the rpm urls presubmit job
// template is rendered with
type rpmURLsPresubmitData struct {
	// Repo is the repository name, e.g. kubevirt
	Repo string
	// Branch is the release branch the job runs on, e.g. release-1.10.
	// It is the plain branch name, like every other job of a release
	// branch: prow matches it as a regular expression, which still
	// matches the branch exactly, while the rehearse plugin passes the
	// string verbatim to git as the ref to clone, where an anchored and
	// quoted regex is an invalid refspec
	Branch string
	// Version is the release version, e.g. 1.10
	Version string
	// BootstrapImage is the container image the job runs
	// make rpm-mirror-checker in, e.g.
	// quay.io/kubevirtci/bootstrap:v20261116-94a0485
	BootstrapImage string
}

// bootstrapImagePrefix is the prefix of the bootstrap images that the
// prow jobs of the kubevirt org run their make targets in
const bootstrapImagePrefix = "quay.io/kubevirtci/bootstrap:"

// bootstrapImage returns the bootstrap image the rpm urls presubmit
// job runs in: the bootstrap image of the repo's presubmits, so that a
// forked branch runs the same image as the jobs it was forked from
// instead of a tag frozen in this repository, which the image bumper
// never walks. There is deliberately no fallback to a pinned tag: every
// job config is expected to hold at least one bootstrap image, and
// without one the append fails rather than rendering an image that no
// automation keeps current. The repo's own presubmits are scanned
// first, the rest of the job config follows in a deterministic order.
func bootstrapImage(jobConfig config.JobConfig, repoKey string) (string, error) {
	scan := func(presubmits []config.Presubmit) (string, bool) {
		for _, presubmit := range presubmits {
			if presubmit.Spec == nil {
				continue
			}
			for _, container := range presubmit.Spec.Containers {
				if strings.HasPrefix(container.Image, bootstrapImagePrefix) {
					return container.Image, true
				}
			}
		}
		return "", false
	}
	if image, ok := scan(jobConfig.PresubmitsStatic[repoKey]); ok {
		return image, nil
	}
	repoKeys := make([]string, 0, len(jobConfig.PresubmitsStatic))
	for key := range jobConfig.PresubmitsStatic {
		repoKeys = append(repoKeys, key)
	}
	sort.Strings(repoKeys)
	for _, key := range repoKeys {
		if key == repoKey {
			continue
		}
		if image, ok := scan(jobConfig.PresubmitsStatic[key]); ok {
			return image, nil
		}
	}
	return "", fmt.Errorf("no presubmit runs in a %s* image", bootstrapImagePrefix)
}

// credentialsRegexp matches credentials embedded in urls, e.g. the github
// token in https://<token>@github.com/...
var credentialsRegexp = regexp.MustCompile(`(https?://)[^@/\s]+@`)

// redactCredentials returns s with credentials embedded in urls replaced
// by ***, so that tokens do not end up in logs
func redactCredentials(s string) string {
	return credentialsRegexp.ReplaceAllString(s, "${1}***@")
}

var configForkerPath = "/usr/bin/config-forker"

func _gitCommand(arg ...string) (string, error) {
	loggedArgs := make([]string, len(arg))
	for i, a := range arg {
		loggedArgs[i] = redactCredentials(a)
	}
	log.Printf("executing 'git %v'", loggedArgs)
	cmd := exec.Command("git", arg...)
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		// ls-remote --exit-code uses status 2 for a normal missing ref.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 2 && slices.Contains(arg, "ls-remote") {
			return "", err
		}
		log.Printf("ERROR: git command output: %s : %s ", redactCredentials(string(bytes)), err)
		return "", err
	}
	return string(bytes), nil
}

func (r *releaseData) generateReleaseNotes() error {
	additionalResources := fmt.Sprintf(`Additional Resources
--------------------
- Mailing list: <https://groups.google.com/forum/#!forum/kubevirt-dev>
- Slack: <https://kubernetes.slack.com/messages/virtualization>
- An easy to use demo: <https://github.com/%s/demo>
- [How to contribute][contributing]
- [License][license]


[contributing]: https://github.com/%s/%s/blob/main/CONTRIBUTING.md
[license]: https://github.com/%s/%s/blob/main/LICENSE
---
`, r.org, r.org, r.repo, r.org, r.repo)

	tagUrl := fmt.Sprintf("https://github.com/%s/%s/releases/tag/%s", r.org, r.repo, r.tag)

	r.releaseNotesFile = fmt.Sprintf("%s/%s-release-notes.txt", r.repoDir, r.tag)

	f, err := os.Create(r.releaseNotesFile)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	// just create releasenotes file and exit if skip is enabled
	if r.skipReleaseNotes {
		return nil
	} else if r.previousTag == "" {
		if r.force {
			log.Printf("Ignoring Release Notes - Unable to generate release notes because no previous tag detected")
			return nil
		} else {
			return fmt.Errorf("unable to generate release notes because no previous tag detected")
		}
	}

	span := fmt.Sprintf("%s..origin/%s", r.previousTag, r.tagBranch)

	fullLogStr, err := gitCommand("-C", r.repoDir, "log", "--oneline", span)
	if err != nil {
		return err
	}

	releaseNotes := []string{}

	fullLogLines := strings.Split(fullLogStr, "\n")
	for _, line := range fullLogLines {
		if strings.Contains(line, "Merge pull request #") {
			pr := strings.Split(line, " ")

			num, err := strconv.Atoi(strings.TrimPrefix(pr[4], "#"))
			if err != nil {
				continue
			}
			note, err := r.getReleaseNote(num)
			if err != nil {
				continue
			}
			if note != "" {
				releaseNotes = append(releaseNotes, note)
			}
		}
	}

	logStr, err := gitCommand("-C", r.repoDir, "log", "--oneline", span)
	if err != nil {
		return err
	}

	contributorStr, err := gitCommand("-C", r.repoDir, "shortlog", "-sne", span)
	if err != nil {
		return err
	}

	contributorList := strings.Split(contributorStr, "\n")

	typeOfChanges, err := gitCommand("-C", r.repoDir, "diff", "--shortstat", span)
	if err != nil {
		return err
	}

	numChanges := strings.Count(logStr, "\n")
	numContributors := len(contributorList)
	typeOfChanges = strings.TrimSpace(typeOfChanges)

	var writeErr error
	writeString := func(s string) {
		if writeErr != nil {
			return
		}
		_, writeErr = f.WriteString(s)
	}

	writeString(fmt.Sprintf("This release follows %s and consists of %d changes, contributed by %d people, leading to %s.\n", r.previousTag, numChanges, numContributors, typeOfChanges))
	if r.promoteRC != "" {
		writeString(fmt.Sprintf("%s is a promotion of release candidate %s which was originally published %s", r.tag, r.promoteRC, r.promoteRCTime.Format("2006-01-02")))
	}
	writeString("\n")
	writeString(fmt.Sprintf("The source code and selected binaries are available for download at: %s.\n", tagUrl))
	writeString("\n")
	writeString("The primary release artifact of KubeVirt is the git tree. The release tag is\n")
	writeString(fmt.Sprintf("signed and can be verified using `git tag -v %s`.\n", r.tag))
	writeString("\n")
	writeString(fmt.Sprintf("Pre-built containers are published on Quay and can be viewed at: <https://quay.io/%s/>.\n", r.org))
	writeString("\n")

	if len(releaseNotes) > 0 {
		writeString("Notable changes\n---------------\n")
		writeString("\n")
		for _, note := range releaseNotes {
			writeString(fmt.Sprintf("- %s\n", note))
		}
	}

	writeString("\n")
	writeString("Contributors\n------------\n")
	writeString(fmt.Sprintf("%d people contributed to this release:\n\n", numContributors))

	for _, contributor := range contributorList {
		if strings.Contains(contributor, "kubevirt-bot") ||
			strings.Contains(contributor, "kubevirt-prow") {
			// skip the bot
			continue
		}
		writeString(fmt.Sprintf("%s\n", strings.TrimSpace(contributor)))
	}

	writeString(additionalResources)
	return writeErr
}

func (r *releaseData) checkoutProjectInfra() error {

	_, err := os.Stat(r.infraDir)
	if err == nil {
		_, err := gitCommand("-C", r.infraDir, "status")
		if err == nil {
			// checkout already exists. default to checkout main
			_, err = gitCommand("-C", r.infraDir, "checkout", "main")
			if err != nil {
				return err
			}

			_, err = gitCommand("-C", r.infraDir, "pull")
			if err != nil {
				return err
			}
			return nil
		}
	}

	// start fresh because checkout doesn't exist or is corrupted
	_ = os.RemoveAll(r.infraDir)
	err = os.MkdirAll(r.infraDir, 0755)
	if err != nil {
		return err
	}

	// add upstream remote branch
	_, err = gitCommand("clone", r.infraUrl, r.infraDir)
	if err != nil {
		return err
	}

	_, err = gitCommand("-C", r.infraDir, "config", "user.name", r.gitUser)
	if err != nil {
		return err
	}

	_, err = gitCommand("-C", r.infraDir, "config", "user.email", r.gitEmail)
	if err != nil {
		return err
	}

	return nil
}

func (r *releaseData) checkoutUpstream() error {
	_, err := os.Stat(r.repoDir)
	if err == nil {
		_, err := gitCommand("-C", r.repoDir, "status")
		if err == nil {
			// checkout already exists. default to checkout main
			_, err = gitCommand("-C", r.repoDir, "checkout", "main")
			if err != nil {
				return err
			}

			_, err = gitCommand("-C", r.repoDir, "pull")
			if err != nil {
				return err
			}
			return nil
		}
	}

	// start fresh because checkout doesn't exist or is corrupted
	_ = os.RemoveAll(r.repoDir)
	err = os.MkdirAll(r.repoDir, 0755)
	if err != nil {
		return err
	}

	// add upstream remote branch
	_, err = gitCommand("clone", r.repoUrl, r.repoDir)
	if err != nil {
		return err
	}

	_, err = gitCommand("-C", r.repoDir, "config", "user.name", r.gitUser)
	if err != nil {
		return err
	}

	_, err = gitCommand("-C", r.repoDir, "config", "user.email", r.gitEmail)
	if err != nil {
		return err
	}
	return nil
}

func (r *releaseData) makeTag(branch string) error {
	if r.promoteRC != "" {
		_, err := gitCommand("-C", r.repoDir, "checkout", r.promoteRC)
		if err != nil {
			return err
		}
	} else {
		_, err := gitCommand("-C", r.repoDir, "checkout", branch)
		if err != nil {
			return err
		}

		_, err = gitCommand("-C", r.repoDir, "pull", "origin", branch)
		if err != nil {
			return err
		}
	}

	if err := r.generateReleaseNotes(); err != nil {
		log.Printf("Warning: release notes generation failed: %v", err)
	}

	_, err := gitCommand("-C", r.repoDir, "tag", "-s", r.tag, "-F", r.releaseNotesFile)
	if err != nil {
		return err
	}
	if !r.dryRun {
		_, err = gitCommand("-C", r.repoDir, "push", r.repoUrl, r.tag)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *releaseData) makeBranch() error {
	_, err := gitCommand("-C", r.repoDir, "checkout", "-b", r.newBranch)
	if err != nil {
		return err
	}

	if !r.dryRun {
		_, err := gitCommand("-C", r.repoDir, "push", r.repoUrl, r.newBranch)
		if err != nil {
			return err
		}
		// make sure to clear cache after successfully creating a new branch
		// This forces the cache to be re-generated if any logic looks
		// at the branches list again. If we don't do this after creating a
		// new branch, then validation logic will fail later on during this
		// execution if we are attempting to cut a branch + new tag from that
		// branch at the same time. It will look like the branch doesn't exist
		// because the cache is outdated. By clearing the cache the branch
		// list will be re-generated.
		r.allBranches = []*github.Branch{}
	}

	return nil
}

func (r *releaseData) getReleaseNote(number int) (string, error) {

	log.Printf("Searching for release note for PR #%d", number)
	pr, _, err := r.githubClient.PullRequests.Get(context.Background(), r.org, r.repo, number)
	if err != nil {
		return "", err
	}

	for _, label := range pr.Labels {
		if label.Name != nil && *label.Name == "release-note-none" {
			return "", nil
		}
	}

	if pr.Body == nil || *pr.Body == "" {
		return "", err
	}

	body := strings.Split(*pr.Body, "\n")

	for i, line := range body {
		if strings.Contains(line, "```release-note") {
			releaseNoteIndex := i + 1
			if len(body) > releaseNoteIndex {
				note := strings.TrimSpace(body[releaseNoteIndex])
				// best effort at fixing some format errors I find
				note = strings.ReplaceAll(note, "\r\n", "")
				note = strings.ReplaceAll(note, "\r", "")
				note = strings.TrimPrefix(note, "- ")
				note = strings.TrimPrefix(note, "-")
				// best effort at catching "none" if the label didn't catch it
				if !strings.Contains(note, "NONE") && strings.ToLower(note) != "none" {
					note = fmt.Sprintf("[PR #%d][%s] %s", number, *pr.User.Login, note)
					return note, nil
				}
			}
		}
	}
	return "", nil
}

func (r *releaseData) projectInfraFork() (string, string, error) {
	owner := r.forkOwner
	ctx := context.Background()
	if owner == "" {
		user, _, err := r.githubClient.Users.Get(ctx, "")
		if err != nil {
			return "", "", fmt.Errorf("looking up project-infra fork owner: %w", err)
		}
		owner = user.GetLogin()
	}
	if strings.EqualFold(owner, "kubevirt") {
		return "", "", fmt.Errorf("project-infra fork owner must not be kubevirt")
	}
	fork, _, err := r.githubClient.Repositories.Get(ctx, owner, "project-infra")
	if err != nil {
		return "", "", fmt.Errorf("accessing %s/project-infra: a writable fork of kubevirt/project-infra is required: %w", owner, err)
	}
	if !fork.GetFork() || (!strings.EqualFold(fork.GetParent().GetFullName(), "kubevirt/project-infra") && !strings.EqualFold(fork.GetSource().GetFullName(), "kubevirt/project-infra")) {
		return "", "", fmt.Errorf("%s/project-infra is not a fork of kubevirt/project-infra", owner)
	}
	if login := fork.GetOwner().GetLogin(); login != "" {
		owner = login
	}
	return owner, fmt.Sprintf("https://%s@github.com/%s/project-infra.git", r.gitToken, owner), nil
}

func (r *releaseData) checkoutProwConfigBranch(branch, forkURL string) error {
	startPoint := "main"
	if forkURL != "" {
		ref := "refs/heads/" + branch
		// With --exit-code, Git returns 2 if the fork branch is absent.
		_, err := gitCommand("-C", r.infraDir, "ls-remote", "--exit-code", "--heads", forkURL, ref)
		if err == nil {
			if _, err := gitCommand("-C", r.infraDir, "fetch", forkURL, ref); err != nil {
				return err
			}
			startPoint = "FETCH_HEAD"
		} else if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 2 {
			return fmt.Errorf("checking for fork branch %s: %w", branch, err)
		}
	}
	_, err := gitCommand("-C", r.infraDir, "checkout", "-B", branch, startPoint)
	return err
}

func (r *releaseData) forkProwJobs() error {
	version := strings.TrimPrefix(r.newBranch, "release-")
	outputConfig := fmt.Sprintf("github/ci/prow-deploy/files/jobs/%s/%s/%s-presubmits-%s.yaml", r.org, r.repo, r.repo, version)
	fullOutputConfig := fmt.Sprintf("%s/%s", r.infraDir, outputConfig)
	fullJobConfig := fmt.Sprintf("%s/github/ci/prow-deploy/files/jobs/%s/%s/%s-presubmits.yaml", r.infraDir, r.org, r.repo, r.repo)

	if _, err := os.Stat(fullJobConfig); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	entries, err := gitCommand("-C", r.infraDir, "ls-tree", "-z", "refs/remotes/origin/main", "--", outputConfig)
	if err != nil {
		return fmt.Errorf("checking for release config on upstream main: %w", err)
	}
	// Git diagnostics on stderr do not contain the NUL terminator of a tree entry.
	if strings.ContainsRune(entries, '\x00') {
		return nil
	}
	if _, err := os.Lstat(fullOutputConfig); err == nil {
		return fmt.Errorf("release config %s exists in the local cache but is not on upstream main; rerun with --clean-cache=true", fullOutputConfig)
	} else if !os.IsNotExist(err) {
		return err
	}

	gitbranch := fmt.Sprintf("%s_%s_%s_configs", r.org, r.repo, r.newBranch)
	var forkOwner, forkURL string
	if !r.dryRun {
		forkOwner, forkURL, err = r.projectInfraFork()
		if err != nil {
			return err
		}
	}
	if err := r.checkoutProwConfigBranch(gitbranch, forkURL); err != nil {
		return err
	}

	// create new prow configs if they don't already exist
	if _, err := os.Stat(fullOutputConfig); err != nil && os.IsNotExist(err) {
		log.Printf("Creating new prow yaml at path %s", fullOutputConfig)
		temp, err := os.MkdirTemp("", "presubmits")
		if err != nil {
			log.Printf("ERROR: temp dir creation failed: %s", err)
			return err
		}
		defer func() {
			if err := os.RemoveAll(temp); err != nil {
				log.Printf("ERROR: removing the temporary presubmits dir %s failed: %s", temp, err)
			}
		}()
		tempPresubmitsConfigPath := filepath.Join(temp, "presubmits.yaml")
		cmd := exec.Command(configForkerPath, "--job-config", fullJobConfig, "--go-version", "1.24", "--version", version, "--output", tempPresubmitsConfigPath)
		bytes, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("ERROR: config-forker command output: %s : %s ", string(bytes), err)
			return err
		}

		// Append the RPM URLs job before writing the final config so an
		// append failure leaves no generated config in the cache.
		_, err = appendRPMURLsJob(tempPresubmitsConfigPath, r.org, r.repo, version)
		if err != nil {
			log.Printf("ERROR: appending the rpm urls presubmit job failed: %s", err)
			return err
		}

		err = configureReleaseJob(tempPresubmitsConfigPath, fullOutputConfig)
		if err != nil {
			log.Printf("ERROR: configuring release job failed: %s", err)
			return err
		}

		_, err = gitCommand("-C", r.infraDir, "add", outputConfig)
		if err != nil {
			return err
		}

		_, err = gitCommand("-C", r.infraDir, "commit", "-s", "-m", fmt.Sprintf("add presubmit job for branch %s", r.newBranch))
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if !r.dryRun {
		if _, err := gitCommand("-C", r.infraDir, "push", forkURL, gitbranch); err != nil {
			return err
		}
		log.Printf("Creating PR for new prow yamls")
		head := fmt.Sprintf("%s:%s", forkOwner, gitbranch)
		ctx := context.Background()
		_, response, err := r.githubClient.PullRequests.Create(ctx, "kubevirt", "project-infra", &github.NewPullRequest{
			Title:               github.String(fmt.Sprintf("Release configs for %s/%s release branch %s", r.org, r.repo, r.newBranch)),
			Body:                github.String("adds new release configs"),
			Head:                github.String(head),
			Base:                github.String("main"),
			MaintainerCanModify: github.Bool(false),
		})
		if err != nil && response != nil && response.StatusCode == http.StatusUnprocessableEntity {
			pulls, _, listErr := r.githubClient.PullRequests.List(ctx, "kubevirt", "project-infra", &github.PullRequestListOptions{State: "open", Head: head, Base: "main"})
			if listErr != nil {
				return listErr
			}
			for _, pull := range pulls {
				if pull.GetState() == "open" && pull.GetHead().GetLabel() == head && pull.GetBase().GetRef() == "main" {
					return nil
				}
			}
		}
		return err
	}

	return nil
}

// configureReleaseJob changes the values from the configuration required for jobs running against
// main branch to what is required in the config for presubmits on release branches. It
//   - changes { run_before_merge: true; always_run: false }
//     to { always_run: true }
//   - deletes { labels.preset_bazel_cache: true }
func configureReleaseJob(configPath string, outputPath string) error {
	if outputPath == configPath {
		return fmt.Errorf("output-path and config-path must not be the same")
	}
	jobConfig, err := config.ReadJobConfig(configPath)
	if err != nil {
		return err
	}
	for key, presubmits := range jobConfig.PresubmitsStatic {
		var newPresubmits []config.Presubmit
		for _, presubmit := range presubmits {
			if presubmit.RunBeforeMerge {
				presubmit.AlwaysRun = true
				presubmit.RunBeforeMerge = false
			}
			delete(presubmit.Labels, "preset-bazel-cache")
			newPresubmits = append(newPresubmits, presubmit)
		}
		jobConfig.PresubmitsStatic[key] = newPresubmits
	}
	marshalled, err := yaml.Marshal(&jobConfig)
	if err != nil {
		return err
	}
	_, err = os.Stat(outputPath)
	if err == nil || !os.IsNotExist(err) {
		return fmt.Errorf("unexpected error on output file %s: %v", outputPath, err)
	}
	err = os.WriteFile(outputPath, marshalled, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

// rpmURLsPresubmit renders the rpm urls presubmit job template for an
// org/repo, a release version, e.g. 1.10, and the bootstrap image the
// job runs make rpm-mirror-checker in. It returns the rendered job
// and the rendered yaml text of the job item, ready to be inserted into a
// job config.
func rpmURLsPresubmit(org, repo, version, bootstrapImage string) (config.Presubmit, string, error) {
	templateData := rpmURLsPresubmitData{
		Repo:           repo,
		Branch:         "release-" + version,
		Version:        version,
		BootstrapImage: bootstrapImage,
	}
	var rendered bytes.Buffer
	if err := rpmURLsPresubmitJobTemplate.Execute(&rendered, templateData); err != nil {
		return config.Presubmit{}, "", fmt.Errorf("failed to render the rpm urls job template: %w", err)
	}
	jobText := rendered.String()

	tempDir, err := os.MkdirTemp("", "rpm-urls-presubmit-*")
	if err != nil {
		return config.Presubmit{}, "", err
	}
	defer func(path string) {
		if err := os.RemoveAll(path); err != nil {
			log.Printf("ERROR: removing the temporary job config dir %s failed: %s", path, err)
		}
	}(tempDir)
	tempPath := filepath.Join(tempDir, "presubmit.yaml")
	// wrap the rendered job item in a job config envelope so that it can
	// be parsed and validated
	envelope := fmt.Sprintf("presubmits:\n  %s/%s:\n%s", org, repo, indentLines(jobText, 2))
	if err := os.WriteFile(tempPath, []byte(envelope), 0666); err != nil {
		return config.Presubmit{}, "", err
	}

	jobConfig, err := config.ReadJobConfig(tempPath)
	if err != nil {
		return config.Presubmit{}, "", fmt.Errorf("the rendered rpm urls job template is not a valid job config: %w", err)
	}
	presubmits, ok := jobConfig.PresubmitsStatic[fmt.Sprintf("%s/%s", org, repo)]
	if !ok || len(presubmits) != 1 {
		return config.Presubmit{}, "", fmt.Errorf("expected exactly one rpm urls presubmit in the template")
	}
	return presubmits[0], jobText, nil
}

// appendRPMURLsJob appends the rpm urls presubmit job to the job config at
// configPath. The job item is inserted into the raw config text so that
// comments and formatting of the existing config are preserved. It returns
// whether the job was added and is a no-op when the job is already present.
func appendRPMURLsJob(configPath, org, repo, version string) (bool, error) {
	jobConfig, err := config.ReadJobConfig(configPath)
	if err != nil {
		return false, err
	}

	repoKey := fmt.Sprintf("%s/%s", org, repo)

	// the job inherits the bootstrap image of the repo's presubmits: the
	// image bumper does not walk releng/, so an image pinned here would
	// never be updated, while the inherited one follows the jobs the
	// branch was forked from
	image, err := bootstrapImage(jobConfig, repoKey)
	if err != nil {
		return false, fmt.Errorf("failed to find a bootstrap image for the rpm urls job in %s: %w", configPath, err)
	}
	job, jobText, err := rpmURLsPresubmit(org, repo, version, image)
	if err != nil {
		return false, err
	}

	presubmits := jobConfig.PresubmitsStatic[repoKey]
	for _, presubmit := range presubmits {
		if presubmit.Name == job.Name {
			log.Printf("job %s is already present in %s, nothing to do", job.Name, configPath)
			return false, nil
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return false, err
	}
	updated, err := insertRPMURLsJob(data, repoKey, jobText)
	if err != nil {
		return false, err
	}

	if err := atomicWriteFile(configPath, updated); err != nil {
		return false, err
	}
	return true, nil
}

// atomicWriteFile replaces the content of path with data via a temporary
// file in the same directory, so that a failed or partial write leaves the
// original file intact
func atomicWriteFile(path string, data []byte) error {
	mode := os.FileMode(0666)
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode()
	}
	temp, err := os.CreateTemp(filepath.Dir(path), fmt.Sprintf(".%s.tmp-*", filepath.Base(path)))
	if err != nil {
		return err
	}
	defer func() {
		// after a successful rename the temporary file is gone: only
		// unexpected removal errors are reported
		if err := os.Remove(temp.Name()); err != nil && !os.IsNotExist(err) {
			log.Printf("ERROR: removing the temporary file %s failed: %s", temp.Name(), err)
		}
	}()
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}

// insertRPMURLsJob inserts the rendered job item for repoKey into the raw
// text of a job config, appended to the existing job list of repoKey. A
// new repoKey entry is created when the config has none yet. Inserting
// text instead of reserializing the config preserves its comments.
func insertRPMURLsJob(data []byte, repoKey, jobText string) ([]byte, error) {
	var doc yamlv3.Node
	if err := yamlv3.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse the job config: %w", err)
	}
	if doc.Kind != yamlv3.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("unexpected job config structure")
	}
	root := doc.Content[0]
	if root.Kind != yamlv3.MappingNode || root.Style&yamlv3.FlowStyle != 0 {
		return nil, fmt.Errorf("unexpected job config structure")
	}

	presubmitsKeyIndex, presubmits := mappingValue(root, "presubmits")
	if presubmits == nil {
		return nil, fmt.Errorf("no presubmits in the job config")
	}
	presubmitsKey := root.Content[presubmitsKeyIndex]

	lines := strings.Split(string(data), "\n")
	ins, err := jobInsertionFor(root, presubmitsKey, presubmits, presubmitsKeyIndex, repoKey, jobText, lines)
	if err != nil {
		return nil, err
	}
	return ins.apply(lines), nil
}

// mappingValue returns the index of the key node of entry in mapping.Content
// and its value node, or -1 and nil when the mapping has no such entry
func mappingValue(mapping *yamlv3.Node, entry string) (int, *yamlv3.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == entry {
			return i, mapping.Content[i+1]
		}
	}
	return -1, nil
}

// nodeLine returns the zero-based line index of node in the text it was
// parsed from: yaml node lines are one-based
func nodeLine(node *yamlv3.Node) int {
	return node.Line - 1
}

// jobInsertion describes a raw text edit that inserts the rendered job
// item into a job config
type jobInsertion struct {
	// replaceIndex is the zero-based index of the line replaced with
	// replaceText, or -1 when no line is replaced
	replaceIndex int
	replaceText  string
	// insertIndex is the zero-based index of the line the insert text is
	// inserted before
	insertIndex int
	// insertText is the text of the inserted job item, already indented
	// to match its surroundings
	insertText string
}

// apply returns the config lines with the insertion applied
func (i jobInsertion) apply(lines []string) []byte {
	if i.replaceIndex >= 0 {
		lines[i.replaceIndex] = i.replaceText
	}
	insertedLines := strings.Split(strings.TrimRight(i.insertText, "\n"), "\n")
	updated := make([]string, 0, len(lines)+len(insertedLines))
	updated = append(updated, lines[:i.insertIndex]...)
	updated = append(updated, insertedLines...)
	updated = append(updated, lines[i.insertIndex:]...)
	return []byte(strings.Join(updated, "\n"))
}

// jobInsertionFor decides how the job item is inserted for repoKey, based
// on the structure of the presubmits section of the job config
func jobInsertionFor(root, presubmitsKey, presubmits *yamlv3.Node, presubmitsKeyIndex int, repoKey, jobText string, lines []string) (jobInsertion, error) {
	switch {
	case presubmits.Kind == yamlv3.MappingNode && len(presubmits.Content) > 0:
		return repoListInsertion(root, presubmits, presubmitsKeyIndex, repoKey, jobText, lines)
	case presubmits.Kind == yamlv3.MappingNode || (presubmits.Kind == yamlv3.ScalarNode && presubmits.Tag == "!!null"):
		// presubmits is null or an empty mapping, e.g. the output of the
		// config-forker for a repo without forkable presubmits
		return emptyPresubmitsInsertion(presubmitsKey, repoKey, jobText), nil
	default:
		return jobInsertion{}, fmt.Errorf("unexpected presubmits structure in the job config")
	}
}

// repoListInsertion returns the insertion for a populated presubmits
// mapping: the job is either appended to the existing job list of repoKey,
// or a new repoKey entry is added when the config has none yet
func repoListInsertion(root, presubmits *yamlv3.Node, presubmitsKeyIndex int, repoKey, jobText string, lines []string) (jobInsertion, error) {
	if presubmits.Style&yamlv3.FlowStyle != 0 {
		return jobInsertion{}, fmt.Errorf("flow style job configs are not supported")
	}
	ins := jobInsertion{
		replaceIndex: -1,
		// insertion point at the end of the job list: right before the
		// next top-level key after presubmits, or at the end of the file
		insertIndex: sectionEndIndex(root, presubmitsKeyIndex, lines),
	}

	repoKeyIndex, jobs := mappingValue(presubmits, repoKey)
	if jobs == nil {
		// the config has presubmits but none for repoKey: add a new entry
		// for it at the end of the presubmits section, indented like the
		// existing repository keys
		keyIndent := presubmits.Content[0].Column - 1
		ins.insertText = fmt.Sprintf("%s%s:\n%s", strings.Repeat(" ", keyIndent), repoKey, indentLines(jobText, keyIndent))
		return ins, nil
	}

	if jobs.Kind != yamlv3.SequenceNode || len(jobs.Content) == 0 {
		return jobInsertion{}, fmt.Errorf("no presubmit jobs for %s in the job config", repoKey)
	}
	if jobs.Style&yamlv3.FlowStyle != 0 {
		return jobInsertion{}, fmt.Errorf("flow style presubmit jobs for %s are not supported", repoKey)
	}
	// append to the job list of repoKey, right before the next repository
	// key when there is one
	if repoKeyIndex+2 < len(presubmits.Content) {
		ins.insertIndex = nodeLine(presubmits.Content[repoKeyIndex+2])
	}
	// the sequence node column is the column of the list dashes
	ins.insertText = indentLines(jobText, jobs.Column-1)
	return ins, nil
}

// emptyPresubmitsInsertion returns the insertion for a presubmits section
// that is null or an empty mapping: the empty value is dropped from the
// presubmits key line and a new entry for repoKey is added under it
func emptyPresubmitsInsertion(presubmitsKey *yamlv3.Node, repoKey, jobText string) jobInsertion {
	keyIndent := presubmitsKey.Column - 1
	return jobInsertion{
		// drop the empty value from the presubmits key line
		replaceIndex: nodeLine(presubmitsKey),
		replaceText:  fmt.Sprintf("%spresubmits:", strings.Repeat(" ", keyIndent)),
		// the new repository key is indented below the presubmits key
		insertIndex: nodeLine(presubmitsKey) + 1,
		insertText:  fmt.Sprintf("%s%s:\n%s", strings.Repeat(" ", keyIndent+2), repoKey, indentLines(jobText, keyIndent+2)),
	}
}

// sectionEndIndex returns the zero-based line index where the section of
// the top-level key at keyIndex ends: right before the line of the next
// top-level key, or at the end of the file. The empty final line produced
// by splitting a newline-terminated file is kept at the end.
func sectionEndIndex(root *yamlv3.Node, keyIndex int, lines []string) int {
	end := len(lines)
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		end--
	}
	if keyIndex+2 < len(root.Content) {
		end = nodeLine(root.Content[keyIndex+2])
	}
	return end
}

// indentLines indents every non-empty line of text by n spaces
func indentLines(text string, n int) string {
	prefix := strings.Repeat(" ", n)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = prefix + line
		}
	}
	return strings.Join(lines, "\n")
}

// addRPMURLsJob adds the rpm urls presubmit job to the job config of an
// existing release branch, e.g. release-1.10, and creates a PR for it.
//
// Branches forked with a release-tool that knows about the job get it
// appended by forkProwJobs already; this is only for branches that were
// forked before the job existed.
func (r *releaseData) addRPMURLsJob(branch string) error {
	version := strings.TrimPrefix(branch, "release-")

	// checkout project infra in order to update the jobs of the branch
	err := r.checkoutProjectInfra()
	if err != nil {
		return err
	}

	outputConfig := fmt.Sprintf("github/ci/prow-deploy/files/jobs/%s/%s/%s-presubmits-%s.yaml", r.org, r.repo, r.repo, version)
	fullOutputConfig := fmt.Sprintf("%s/%s", r.infraDir, outputConfig)

	// the job config of the branch is created when the branch is forked
	if _, err := os.Stat(fullOutputConfig); err != nil && os.IsNotExist(err) {
		return fmt.Errorf("no job config at %s, was the release branch forked?", outputConfig)
	}

	// discard uncommitted leftovers of an aborted run, from the working
	// tree and the index: a modified but uncommitted config must not make
	// the append below report the job as already present on main. A
	// failed restore must stop the backfill: continuing would risk
	// pushing a commit that does not contain the job.
	if _, err := gitCommand("-C", r.infraDir, "checkout", "HEAD", "--", outputConfig); err != nil {
		return fmt.Errorf("failed to restore the job config %s from main: %w", outputConfig, err)
	}

	gitbranch := fmt.Sprintf("%s_%s_%s_rpm_urls_job", r.org, r.repo, branch)

	// Reuse the topic branch of a previous run when it exists, so that a
	// retry, e.g. after a failed PR creation, does not rewrite already
	// pushed history with a new commit. The branch is taken from the
	// remote when it is not available locally.
	if _, err := gitCommand("-C", r.infraDir, "checkout", gitbranch); err != nil {
		if _, fetchErr := gitCommand("-C", r.infraDir, "fetch", r.infraUrl, gitbranch); fetchErr == nil {
			_, err = gitCommand("-C", r.infraDir, "checkout", "-B", gitbranch, "FETCH_HEAD")
		} else {
			_, err = gitCommand("-C", r.infraDir, "checkout", "-b", gitbranch)
		}
		if err != nil {
			return err
		}
	}

	added, err := appendRPMURLsJob(fullOutputConfig, r.org, r.repo, version)
	if err != nil {
		return err
	}

	if added {
		_, err = gitCommand("-C", r.infraDir, "add", outputConfig)
		if err != nil {
			return err
		}

		// commit only the job config, so that unrelated files staged in a
		// reused cache cannot end up in the backfill commit
		_, err = gitCommand("-C", r.infraDir, "commit", "-s", "-m", fmt.Sprintf("add rpm urls presubmit job for branch %s", branch), "--", outputConfig)
		if err != nil {
			return err
		}
	}

	if !r.dryRun {
		_, err = gitCommand("-C", r.infraDir, "push", r.infraUrl, gitbranch)
		if err != nil {
			return err
		}

		log.Printf("Creating PR for the rpm urls job")
		cmd := exec.Command("/usr/bin/pr-creator",
			"--org", "kubevirt",
			"--repo", "project-infra",
			"--branch", "main",
			"--github-token-path", r.githubTokenPath,
			"--title", fmt.Sprintf("Add rpm urls presubmit job for %s/%s release branch %s", r.org, r.repo, branch),
			"--body", fmt.Sprintf("adds the rpm urls presubmit job for branch %s", branch),
			"--source", fmt.Sprintf("kubevirt:%s", gitbranch),
			"--confirm",
		)
		bytes, err := cmd.CombinedOutput()
		if err != nil && !strings.Contains(string(bytes), "A pull request already exists") {
			log.Printf("ERROR: pr-creator command output: %s : %s ", redactCredentials(string(bytes)), err)
			return err
		}
	}

	return nil
}

func (r *releaseData) cutNewBranch(skipProw bool) error {
	if !skipProw {
		// checkout project infra project in order to update jobs for new branch
		err := r.checkoutProjectInfra()
		if err != nil {
			return err
		}

		err = r.forkProwJobs()
		if err != nil {
			return err
		}
	}

	// checkout remote branch
	err := r.checkoutUpstream()
	if err != nil {
		return err
	}

	err = r.makeBranch()
	if err != nil {
		return err
	}

	return nil
}

func (r *releaseData) isRCInvalid(release *github.RepositoryRelease, branch string) (bool, bool, error) {
	isInvalid := false
	blocksNewRC := false
	releaseTime := release.CreatedAt.Time

	entry, err := r.getBlockers(branch)
	if err != nil {
		return false, false, err
	}

	// Check for blocker issues, both closed and open.
	// An open issue blocks the promotion of an RC
	// An issue that has the blocker lable which gets closed after
	// the RC was cut will result in a new RC being made.

	// print the issues to give feedback about what is blocking the release
	for _, issue := range entry.allBlockerIssues {
		if *issue.State == "open" {
			log.Printf("RC Promotion blocked [issue #%d - %s] %s", *issue.Number, *issue.URL, *issue.Title)
			isInvalid = true
			blocksNewRC = true
		} else if issue.ClosedAt != nil && issue.ClosedAt.After(releaseTime) {
			// if a blocker issue was closed after the release was cut,
			// then we can't promote the RC, instead we have to cut a new RC.
			log.Printf("RC Promotion invalidated by [issue #%d - %s] %s", *issue.Number, *issue.URL, *issue.Title)
			isInvalid = true
		}
	}

	// Check if any blocker PRs exist for the branch
	for _, pr := range entry.allBlockerPRs {
		if pr.ClosedAt == nil || *pr.State == "open" {
			log.Printf("BLOCKED BY [PR #%d - %s] %s", *pr.Number, *pr.URL, *pr.Title)
			isInvalid = true
			blocksNewRC = true
		} else if pr.ClosedAt.After(releaseTime) {
			log.Printf("RC Invalidated by [PR #%d - %s] %s", *pr.Number, *pr.URL, *pr.Title)
			isInvalid = true
		}
	}

	return isInvalid, blocksNewRC, nil
}

func (r *releaseData) isBranchBlocked(branch string) (bool, error) {
	prBlocked, err := r.hasOpenBlockerPRs(branch)

	if err != nil {
		return false, err
	}

	issueBlocked, err := r.hasOpenBlockerIssues(branch)
	if err != nil {
		return false, err
	}

	if issueBlocked || prBlocked {
		if r.force {
			log.Printf("Ignoring blockers to to use of [--force] option")
			return false, nil
		}
		return true, nil
	}

	log.Printf("no blockers found")
	return false, nil
}

func (r *releaseData) hasOpenBlockerPRs(branch string) (bool, error) {

	entry, err := r.getBlockers(branch)
	if err != nil {
		return false, err
	}

	hasOpenBlockers := false
	// print the issues to give feedback about what is blocking the release
	for _, pr := range entry.allBlockerPRs {
		if pr.State != nil && *pr.State == "open" {
			log.Printf("BLOCKED BY [PR #%d - %s] %s", *pr.Number, *pr.URL, *pr.Title)
			hasOpenBlockers = true
		}
	}

	return hasOpenBlockers, nil
}

func (r *releaseData) hasOpenBlockerIssues(branch string) (bool, error) {

	entry, err := r.getBlockers(branch)
	if err != nil {
		return false, err
	}

	hasOpenBlocker := false
	// print the issues to give feedback about what is blocking the release
	for _, issue := range entry.allBlockerIssues {
		if issue.State != nil && *issue.State == "open" {
			log.Printf("BLOCKED BY [issue #%d - %s] %s", *issue.Number, *issue.URL, *issue.Title)
			hasOpenBlocker = true
		}
	}

	return hasOpenBlocker, nil
}

func (r *releaseData) verifyPromoteRC() error {
	if r.tag != "" {
		return fmt.Errorf("--new-release and --promote-rc can not be used together. --promote-rc detects the correct tag to make an official release out of")
	}

	re := regexp.MustCompile(`^v\d*\.\d*.\d*-rc.\d*$`)
	match := re.FindString(r.promoteRC)
	if match == "" {
		return fmt.Errorf("--promote-rc=%s is invalid.  must point to a release candidate tag in the form of v[x].[y].[z]-rc.[n]. Example v0.31.0-rc.1 is valid and will result in the promotion of official v0.31.0 release", r.promoteRC)
	}

	tagSemver, err := semver.NewVersion(match)
	if err != nil {
		return err
	}

	r.tag = fmt.Sprintf("v%d.%d.%d", tagSemver.Major(), tagSemver.Minor(), tagSemver.Patch())
	log.Printf("promoting rc [%s] as tag [%s]", r.promoteRC, r.tag)

	return nil
}

func (r *releaseData) doesBranchExist(branch string) (bool, error) {

	branches, err := r.getBranches()
	if err != nil {
		return false, err
	}

	for _, b := range branches {
		if b.Name != nil && *b.Name == branch {
			return true, nil
		}
	}

	return false, nil
}

// releaseBranchRegexp matches release branch names, i.e. release-x.y.
// It is compiled once and reused for every check.
var releaseBranchRegexp = regexp.MustCompile(`^release-\d*\.\d*$`)

// isValidReleaseBranch reports whether branch is formatted as a release
// branch, i.e. release-x.y
func isValidReleaseBranch(branch string) bool {
	return releaseBranchRegexp.MatchString(branch)
}

func (r *releaseData) verifyBranch() error {

	exists, err := r.doesBranchExist(r.newBranch)
	if err != nil {
		return err
	} else if exists {
		log.Printf("Release branch [%s] already exists", r.newBranch)
		return nil
	}

	if !isValidReleaseBranch(r.newBranch) {
		return fmt.Errorf("malformed release branch name [%s]. Branch name must be formatted as release-[x].[y]. For example a branch for release 0.30.0 would be release-0.30", r.newBranch)
	}

	return nil
}

func (r *releaseData) getBlockers(branch string) (*blockerListCacheEntry, error) {

	blockerLabel := fmt.Sprintf("release-blocker/%s", branch)

	cache, ok := r.blockerListCache[blockerLabel]
	if ok {
		return cache, nil
	}

	issueListOptions := &github.IssueListByRepoOptions{
		// filtering by labels here gives inconsistent results
		// with the github api, so we double check that the blocker
		// label exists as well.
		Labels: []string{blockerLabel},
		ListOptions: github.ListOptions{
			PerPage: 10000,
		},
	}
	prListOptions := &github.PullRequestListOptions{
		Base: branch,
		ListOptions: github.ListOptions{
			PerPage: 10000,
		},
	}

	prListOptions.State = "all"
	issueListOptions.State = "all"
	if branch == "main" {
		// there's never a reason to list all PRs/Issues (both open and closed) in the entire project for main
		// We do care about open and closed PRS for stable branches though
		prListOptions.State = "open"
		issueListOptions.State = "open"
	}

	issues, _, err := r.githubClient.Issues.ListByRepo(context.Background(), r.org, r.repo, issueListOptions)
	if err != nil {
		return nil, err
	}

	prs, _, err := r.githubClient.PullRequests.List(context.Background(), r.org, r.repo, prListOptions)
	if err != nil {
		return nil, err
	}

	filteredPRs := []*github.PullRequest{}
	filteredIssues := []*github.Issue{}

	for _, pr := range prs {
		if pr.Labels == nil {
			continue
		}
		for _, label := range pr.Labels {
			if label.Name != nil && *label.Name == blockerLabel {
				filteredPRs = append(filteredPRs, pr)
				break
			}
		}
	}

	for _, issue := range issues {
		if issue.Labels == nil {
			continue
		}
		for _, label := range issue.Labels {
			if label.Name != nil && *label.Name == blockerLabel {
				filteredIssues = append(filteredIssues, issue)
				break
			}
		}
	}

	r.blockerListCache[blockerLabel] = &blockerListCacheEntry{
		allBlockerIssues: filteredIssues,
		allBlockerPRs:    filteredPRs,
	}

	return r.blockerListCache[blockerLabel], nil
}

func (r *releaseData) getBranches() ([]*github.Branch, error) {

	if len(r.allBranches) != 0 {
		return r.allBranches, nil
	}

	// github caps per_page at 100: follow the pagination until all
	// branches were seen, a single response is not enough to decide that
	// a branch does not exist
	var allBranches []*github.Branch
	opt := &github.BranchListOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	for {
		branches, resp, err := r.githubClient.Repositories.ListBranches(context.Background(), r.org, r.repo, opt)
		if err != nil {
			return nil, err
		}
		allBranches = append(allBranches, branches...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	r.allBranches = allBranches

	return r.allBranches, nil

}

func (r *releaseData) getReleases() ([]*github.RepositoryRelease, error) {

	if len(r.allReleases) != 0 {
		return r.allReleases, nil
	}

	releases, _, err := r.githubClient.Repositories.ListReleases(context.Background(), r.org, r.repo, &github.ListOptions{PerPage: 10000})

	if err != nil {
		return nil, err
	}
	r.allReleases = releases

	return r.allReleases, nil
}

func (r *releaseData) autoDetectData(autoReleaseCadance string, autoPromoteAfterDays int) error { //nolint:gocyclo

	log.Printf("Attempting to auto detect release for %s/%s", r.org, r.repo)

	releaseDaily := false
	releaseMonthly := false
	shouldMakeNewMinorRelease := false
	shouldMakeNewRC := false
	shouldPromoteRC := false

	if autoReleaseCadance == "daily" {
		releaseDaily = true
	} else if autoReleaseCadance == "monthly" {
		releaseMonthly = true
	} else {
		return fmt.Errorf("Unknown cadance [%s]", autoReleaseCadance)
	}

	now := r.now
	releases, err := r.getReleases()
	if err != nil {
		return err
	}

	// Auto logic sequence of events
	// 0. Find all official releases, and sort
	// 1. Detect what the next x.y.0 release should be
	// 2. Detect if it's time to cut the next x.y.0 release series
	// 3. Find the most recent RC for the next release series.
	// 5. detect if RC is still valid and no blockers occurred
	//    - scan RC creation time
	//    - scan for PRs or ISSUES both closed or open which have blocker label
	//    - if blocker label was set after the rc was cut, invalidate the RC
	// 6. Cut new RC if last RC is invalid due to blockers
	// 7. Detect if it's time to promote a x.y.0.rc.n release to an official release.

	// ---------------------------------------
	// 0. Find all official releases, and sort
	// ---------------------------------------
	var vs []*semver.Version
	for _, release := range releases {
		if (release.Draft != nil && *release.Draft) ||
			(release.Prerelease != nil && *release.Prerelease) ||
			strings.Contains(*release.TagName, "-rc.") ||
			len(release.Assets) == 0 {

			continue
		}

		v, err := semver.NewVersion(*release.TagName)
		if err != nil {
			// not an official release if it's not semver compatiable.
			continue
		}
		vs = append(vs, v)
	}

	// decending order from most recent.
	sort.Sort(sort.Reverse(semver.Collection(vs)))

	// -----------------------------------------------
	// 1. Detect what the next x.y.0 release should be
	// -----------------------------------------------
	nextMinorRelease := ""
	nextMinorReleaseRC := ""
	nextMinorReleaseBranch := ""
	currentMinorRelease := ""

	for _, v := range vs {
		if v.Patch() == 0 {
			currentMinorRelease = fmt.Sprintf("v%d.%d.0", v.Major(), v.Minor())
			nextMinorRelease = fmt.Sprintf("v%d.%d.0", v.Major(), v.Minor()+1)
			nextMinorReleaseRC = fmt.Sprintf("v%d.%d.0-rc.0", v.Major(), v.Minor()+1)
			nextMinorReleaseBranch = fmt.Sprintf("release-%d.%d", v.Major(), v.Minor()+1)

			log.Printf("Last Minor Release Series: v%d.%d", v.Major(), v.Minor())
			log.Printf("Next Minor Release Series: v%d.%d", v.Major(), v.Minor()+1)
			break
		}
	}

	// -----------------------------------------------------------
	// 2. Detect if it's time to cut the next x.y.0 release series
	// -----------------------------------------------------------
	for _, release := range releases {
		if *release.TagName == nextMinorReleaseRC || *release.TagName == nextMinorRelease {
			// new release already exists
			shouldMakeNewMinorRelease = false
			break
		} else if *release.TagName == currentMinorRelease {
			createdAt := *release.CreatedAt

			if releaseDaily &&
				now.Day() > createdAt.Time.Day() {
				shouldMakeNewMinorRelease = true
			} else if (releaseDaily || releaseMonthly) &&
				now.Month() > createdAt.Time.Month() {
				shouldMakeNewMinorRelease = true
			} else if (releaseDaily || releaseMonthly) &&
				now.Year() > createdAt.Time.Year() {
				shouldMakeNewMinorRelease = true
			}
		}
	}

	// -------------------------------------------------------
	// 3. Find the most recent RC for the next release series.
	// -------------------------------------------------------
	highestRC := 0
	var rcPromotionCandidate *github.RepositoryRelease
	rcTemplate := fmt.Sprintf("%s-rc.", nextMinorRelease)
	for _, release := range releases {

		if *release.TagName == nextMinorRelease {
			// found official release, so rc is already already promoted
			rcPromotionCandidate = nil
			highestRC = 0
			break
		} else if !strings.Contains(*release.TagName, rcTemplate) {
			// not an rc for this release
			continue
		}

		rcNumber, err := strconv.Atoi(strings.TrimPrefix(*release.TagName, rcTemplate))
		if err != nil {
			continue
		} else if rcNumber >= highestRC {
			highestRC = rcNumber
			rcPromotionCandidate = release
		}
	}

	if rcPromotionCandidate != nil {
		log.Printf("Most recent RC for next release series is detected as %s", *rcPromotionCandidate.TagName)
		releaseTime := rcPromotionCandidate.CreatedAt.Time

		// -------------------------------------------------------
		// 5. detect if RC is still valid and no blockers occurred
		// -------------------------------------------------------
		invalidated, blocksNewRC, err := r.isRCInvalid(rcPromotionCandidate, nextMinorReleaseBranch)
		if err != nil {
			return err
		}

		// 86400 seconds in a day
		// we're giving it a tolarance of a couple of hours
		// in order to account for a periodic being delayed
		// slightly when cutting the RC, which without a slight
		// tolerance would cause the periodic that should technically
		// do the release to wait an additional day due to the periodic
		// only running once dailing.
		// 3 hour tolerance.
		tolerance := int64(3 * 60 * 60)
		promoteAfterSeconds := int64(86400*autoPromoteAfterDays) - tolerance
		secondsDiff := now.UTC().Unix() - releaseTime.UTC().Unix()

		if invalidated {
			// ----------------------------------------------------------------------------------
			// 6. Cut new RC if last RC is invalid due to blockers and those blockers are resolved
			// -----------------------------------------------------------------------------------
			if !blocksNewRC {
				highestRC++
				nextMinorReleaseRC = fmt.Sprintf("%s%d", rcTemplate, highestRC)
				shouldMakeNewRC = true
				log.Printf("Cutting new RC due to blocker %s", nextMinorReleaseRC)
			}
		} else if secondsDiff >= promoteAfterSeconds {

			// ------------------------------------------------------------------------------
			// 7. Detect if it's time to promote a x.y.0.rc.n release to an official release.
			// ------------------------------------------------------------------------------

			shouldPromoteRC = true
		} else {
			log.Printf("Waiting to promote RC %s. %d seconds remain", *rcPromotionCandidate.TagName, promoteAfterSeconds-secondsDiff)
		}
	}

	if shouldMakeNewMinorRelease {
		log.Printf("Auto mode detected a new branch [%s] and new tag [%s] should be created", nextMinorReleaseBranch, nextMinorReleaseRC)

		r.tag = nextMinorReleaseRC
		r.newBranch = nextMinorReleaseBranch
		r.tagBranch = nextMinorReleaseBranch

		return nil
	} else if shouldMakeNewRC {
		log.Printf("Auto creating new RC [%s] for branch [%s]", nextMinorReleaseRC, nextMinorReleaseBranch)

		r.tag = nextMinorReleaseRC
		r.tagBranch = nextMinorReleaseBranch

		return nil
	} else if shouldPromoteRC {
		log.Printf("Auto promoting rc [%s] after [%d] days", *rcPromotionCandidate.TagName, autoPromoteAfterDays)
		r.promoteRC = *rcPromotionCandidate.TagName
		return nil
	}

	log.Printf("No auto action detected for %s/%s", r.org, r.repo)
	return nil
}

func (r *releaseData) verifyTag() error { //nolint:gocyclo
	// must be a valid semver version
	tagSemver, err := semver.NewVersion(r.tag)
	if err != nil {
		return err
	}

	expectedBranch := r.tagBranch
	if expectedBranch == "" {
		expectedBranch = fmt.Sprintf("release-%d.%d", tagSemver.Major(), tagSemver.Minor())
	}

	releases, err := r.getReleases()
	if err != nil {
		return err
	}
	for _, release := range releases {
		if *release.TagName == r.tag {
			log.Printf("Release tag [%s] already exists", r.tag)
			return nil
		}
	}

	// ensure if promoteRC tag exists that it is eligible for promotion
	if r.promoteRC != "" {
		found := false
		invalid := false
		for _, release := range releases {
			if *release.TagName == r.promoteRC {
				found = true
				invalid, _, err = r.isRCInvalid(release, expectedBranch)
				if err != nil {
					return err
				}
				r.promoteRCTime = release.CreatedAt.Time
				break
			}
		}
		if !found {
			return fmt.Errorf("Unable to find promote-rc tag [%s]", r.promoteRC)
		} else if invalid {
			return fmt.Errorf("RC [%s] is ineligible to be promoted due to blockers.", r.promoteRC)
		}
	} else {
		// if this is not a promotion, ensure the release is either an RC or a patch release.
		re := []*regexp.Regexp{
			regexp.MustCompile(`^v\d*\.\d*.\d*-rc.\d*$`),
			regexp.MustCompile(`^v\d*\.\d*.\d*-alpha.\d*$`),
			regexp.MustCompile(`^v\d*\.\d*.\d*-beta.\d*$`),
		}

		match := ""
		for _, matcher := range re {
			match = matcher.FindString(r.tag)
			if match != "" {
				break
			}
		}

		if match == "" && tagSemver.Patch() == 0 && !r.force {
			return fmt.Errorf("The tag [%s] must be release candidate or alpha since it is not being promoted from another tag.", r.tag)
		}

	}

	var vs []*semver.Version

	for _, release := range releases {
		if (release.Draft != nil && *release.Draft) ||
			(release.Prerelease != nil && *release.Prerelease) ||
			len(release.Assets) == 0 {

			continue
		}
		v, err := semver.NewVersion(*release.TagName)
		if err != nil {
			// not an official release if it's not semver compatiable.
			continue
		}
		vs = append(vs, v)
	}

	// decending order from most recent.
	sort.Sort(sort.Reverse(semver.Collection(vs)))

	for _, v := range vs {
		if v.LessThan(tagSemver) {
			r.previousTag = fmt.Sprintf("v%v", v)
			break
		}
	}

	if r.previousTag == "" {
		log.Printf("No previous release tag found for tag [%s]", r.tag)
	} else {
		log.Printf("Previous Tag [%s]", r.previousTag)
	}

	branches, err := r.getBranches()
	if err != nil {
		return err
	}

	if r.tagBranch == "" {
		var releaseBranch *github.Branch
		for _, branch := range branches {
			if branch.Name != nil && *branch.Name == expectedBranch {
				releaseBranch = branch
				break
			}
		}

		if releaseBranch == nil {
			return fmt.Errorf("release branch [%s] not found for new release [%s]", expectedBranch, r.tag)
		}

		r.tagBranch = expectedBranch
	}
	return nil
}

func (r *releaseData) cutNewTag() error {

	// checkout remote branch
	err := r.checkoutUpstream()
	if err != nil {
		return err
	}

	return r.makeTag(r.tagBranch)
}

func (r *releaseData) printData() {

	if r.dryRun {
		log.Print("DRY-RUN")
	}

	log.Print("Input Data")

	log.Printf("\trepoUrl: %s", redactCredentials(r.repoUrl))
	log.Printf("\tnewTag: %s", r.tag)
	log.Printf("\tnewBranch: %s", r.newBranch)
	log.Printf("\texistingBranch: %s", r.tagBranch)
	log.Printf("\torg: %s", r.org)
	log.Printf("\trepo: %s", r.repo)
	log.Printf("\tforce: %t", r.force)
	log.Printf("\tgithubTokenFile: %s", r.githubTokenPath)
}

// addRPMURLsJobCommand runs the --add-rpm-urls-job dispatch: it validates
// the requested branch and adds the rpm urls presubmit job to it
func addRPMURLsJobCommand(r *releaseData, branch string) {
	if r.newBranch != "" || r.tag != "" {
		log.Fatal("ERROR --add-rpm-urls-job cannot be combined with --new-branch or --new-release")
	}

	if !isValidReleaseBranch(branch) {
		log.Fatalf("ERROR malformed release branch name [%s]. Branch name must be formatted as release-[x].[y]. For example a branch for release 0.30.0 would be release-0.30", branch)
	}

	exists, err := r.doesBranchExist(branch)
	if err != nil {
		log.Fatalf("ERROR Invalid branch: %s ", err)
	}
	if !exists {
		log.Fatalf("ERROR Release branch [%s] does not exist", branch)
	}

	if err := r.addRPMURLsJob(branch); err != nil {
		log.Fatalf("ERROR Adding the rpm urls job: %s ", err)
	}
}

func main() {
	newBranch := flag.String("new-branch", "", "New branch to cut from main.")
	existingBranch := flag.String("existing-branch", "", "Cut the tag from an existing branch.")
	releaseTag := flag.String("new-release", "", "New release tag. Must be a valid semver. The branch is automatically detected from the major and minor release")
	org := flag.String("org", "", "The project org")
	repo := flag.String("repo", "", "The project repo")
	dryRun := flag.Bool("dry-run", true, "Should this be a dry run")
	cacheDir := flag.String("cache-dir", "/tmp/release-tool", "The base directory used to cache git repos in")
	cleanCacheDir := flag.Bool("clean-cache", true, "Clean the cache dir before executing")
	githubTokenFile := flag.String("github-token-file", "", "file containing the github token.")
	gitUser := flag.String("git-user", "", "git user")
	gitEmail := flag.String("git-email", "", "git user email")
	skipReleaseNotes := flag.Bool("skip-release-notes", false, "skip generating release notes for a tag")
	force := flag.Bool("force", false, "force a release or release branch to occur despite blockers or other warnings")
	skipProw := flag.Bool("skip-prow", false, "skip creating prow configs")
	addRPMURLsJob := flag.String("add-rpm-urls-job", "", "Add the rpm urls presubmit job to the job config of an existing release branch, e.g. release-1.10")
	forkOwner := flag.String("fork-owner", "", "Owner of the project-infra fork for Prow config PRs (defaults to the authenticated user)")
	promoteRC := flag.String("promote-release-candidate", "", "The tag of an rc release that will be promoted to an official release")

	autoRelease := flag.Bool("auto-release", false, "Automatically perform branch cutting an releases based on time intervals")
	autoReleaseCadance := flag.String("auto-release-cadance", "monthly", "set the auto release cadance to daily or monthly")
	autoPromoteAfterDays := flag.Int("auto-promote-after-days", 7, "Set the set the time before autopromoting a release candidate")

	flag.Parse()

	if *org == "" {
		log.Fatal("--org is a required argument")
	} else if *repo == "" {
		log.Fatal("--repo is a required argument")
	} else if *gitUser == "" {
		log.Fatal(" --git-user is required")
	} else if *gitEmail == "" {
		log.Fatal("--git-email is required")
	} else if *githubTokenFile == "" {
		log.Fatal("--github-token-file is a required argument")
	}

	tokenBytes, err := os.ReadFile(*githubTokenFile)
	if err != nil {
		log.Fatalf("ERROR accessing github token: %s ", err)
	}
	token := strings.TrimSpace(string(tokenBytes))

	repoUrl := fmt.Sprintf("https://%s@github.com/%s/%s.git", token, *org, *repo)
	infraUrl := fmt.Sprintf("https://%s@github.com/kubevirt/project-infra.git", token)
	repoDir := fmt.Sprintf("%s/%s/https-%s", *cacheDir, *org, *repo)
	infraDir := fmt.Sprintf("%s/%s/https-%s", *cacheDir, "kubevirt", "project-infra")

	if *cleanCacheDir {
		_ = os.RemoveAll(repoDir)
		_ = os.RemoveAll(infraDir)
	}

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	r := releaseData{
		repoDir:          repoDir,
		infraDir:         infraDir,
		repoUrl:          repoUrl,
		infraUrl:         infraUrl,
		repo:             *repo,
		org:              *org,
		newBranch:        *newBranch,
		forkOwner:        *forkOwner,
		tag:              *releaseTag,
		tagBranch:        *existingBranch,
		promoteRC:        *promoteRC,
		skipReleaseNotes: *skipReleaseNotes,

		gitUser:  *gitUser,
		gitEmail: *gitEmail,
		gitToken: token,

		githubClient:    client,
		githubTokenPath: *githubTokenFile,
		dryRun:          *dryRun,
		force:           *force,

		blockerListCache: make(map[string]*blockerListCacheEntry),

		now: time.Now(),
	}

	if *autoRelease {
		if err := r.autoDetectData(*autoReleaseCadance, *autoPromoteAfterDays); err != nil {
			log.Fatalf("ERROR during auto-detect: %v", err)
		}
	}

	// If this is a promotion, we need to set the tag to promote
	if r.promoteRC != "" {
		// verifies promotion RC is valid and sets the expected new tag
		err := r.verifyPromoteRC()
		if err != nil {
			log.Fatalf("ERROR during promotion validation: %v", err)
		}
	}

	r.printData()

	if *addRPMURLsJob != "" {
		addRPMURLsJobCommand(&r, *addRPMURLsJob)
		return
	}

	if r.newBranch != "" {
		err := r.verifyBranch()
		if err != nil {
			log.Fatalf("ERROR Invalid branch: %s ", err)
		}

		blocked, err := r.isBranchBlocked("main")
		if err != nil {
			log.Fatalf("ERROR retreiving blockers for branch Branch: %s ", err)
		} else if blocked {
			log.Fatal("ERROR Branch is blocked")
		}

		exists, err := r.doesBranchExist(r.newBranch)
		if err != nil {
			log.Fatalf("ERROR Invalid branch: %s ", err)
		}

		if !exists {
			err = r.cutNewBranch(*skipProw)
			if err != nil {
				log.Fatalf("ERROR Creating Branch: %s ", err)
			}
		}
	}

	if r.tag != "" {
		// make sure the tag is valid
		// this also sets the tag branch as expected
		err := r.verifyTag()
		if err != nil {
			log.Fatalf("ERROR Invalid Tag: %s ", err)
		}

		blocked, err := r.isBranchBlocked(r.tagBranch)
		if err != nil {
			log.Fatalf("ERROR retreiving blockers for branch Branch: %s ", err)
		} else if blocked {
			log.Fatalf("ERROR Branch %s is blocked ", r.tagBranch)
		}

		err = r.cutNewTag()
		if err != nil {
			log.Fatalf("ERROR Creating Tag: %s ", err)
		}
	}
}

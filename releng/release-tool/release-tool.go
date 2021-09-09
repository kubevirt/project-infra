package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"golang.org/x/oauth2"

	"github.com/Masterminds/semver"
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

func _gitCommand(arg ...string) (string, error) {
	log.Printf("executing 'git %v", arg)
	cmd := exec.Command("git", arg...)
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ERROR: git command output: %s : %s ", string(bytes), err)
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
	defer f.Close()

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

	f.WriteString(fmt.Sprintf("This release follows %s and consists of %d changes, contributed by %d people, leading to %s.\n", r.previousTag, numChanges, numContributors, typeOfChanges))
	if r.promoteRC != "" {
		f.WriteString(fmt.Sprintf("%s is a promotion of release candidate %s which was originally published %s", r.tag, r.promoteRC, r.promoteRCTime.Format("2006-01-02")))
	}
	f.WriteString("\n")
	f.WriteString(fmt.Sprintf("The source code and selected binaries are available for download at: %s.\n", tagUrl))
	f.WriteString("\n")
	f.WriteString("The primary release artifact of KubeVirt is the git tree. The release tag is\n")
	f.WriteString(fmt.Sprintf("signed and can be verified using `git tag -v %s`.\n", r.tag))
	f.WriteString("\n")
	f.WriteString(fmt.Sprintf("Pre-built containers are published on Quay and can be viewed at: <https://quay.io/%s/>.\n", r.org))
	f.WriteString("\n")

	if len(releaseNotes) > 0 {
		f.WriteString("Notable changes\n---------------\n")
		f.WriteString("\n")
		for _, note := range releaseNotes {
			f.WriteString(fmt.Sprintf("- %s\n", note))
		}
	}

	f.WriteString("\n")
	f.WriteString("Contributors\n------------\n")
	f.WriteString(fmt.Sprintf("%d people contributed to this release:\n\n", numContributors))

	for _, contributor := range contributorList {
		if strings.Contains(contributor, "kubevirt-bot") {
			// skip the bot
			continue
		}
		f.WriteString(fmt.Sprintf("%s\n", strings.TrimSpace(contributor)))
	}

	f.WriteString(additionalResources)
	return nil
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
	os.RemoveAll(r.infraDir)
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
	os.RemoveAll(r.repoDir)
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

	r.generateReleaseNotes()

	_, err := gitCommand("-C", r.repoDir, "tag", "-s", r.tag, "-F", r.releaseNotesFile)

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

func (r *releaseData) forkProwJobs() error {
	version := strings.TrimPrefix(r.newBranch, "release-")
	outputConfig := fmt.Sprintf("github/ci/prow-deploy/files/jobs/%s/%s/%s-presubmits-%s.yaml", r.org, r.repo, r.repo, version)
	fullOutputConfig := fmt.Sprintf("%s/%s", r.infraDir, outputConfig)
	fullJobConfig := fmt.Sprintf("%s/github/ci/prow-deploy/files/jobs/%s/%s/%s-presubmits.yaml", r.infraDir, r.org, r.repo, r.repo)

	gitbranch := fmt.Sprintf("%s_%s_%s_configs", r.org, r.repo, r.newBranch)

	_, err := gitCommand("-C", r.infraDir, "checkout", "-b", gitbranch)
	if err != nil {
		_, err = gitCommand("-C", r.infraDir, "checkout", "-B", gitbranch)
		if err != nil {
			return err
		}

	}
	// ignore error here, we're just trying to make sure we've synced
	// and pulled down any changes from the origin branch in github
	// in case there are non local changes that need to get pulled in.
	_, _ = gitCommand("-C", r.infraDir, "pull", "origin", gitbranch)

	if _, err = os.Stat(fullJobConfig); err != nil && os.IsNotExist(err) {
		// no job to fork for this project
		return nil
	}

	// create new prow configs if they don't already exist
	if _, err := os.Stat(fullOutputConfig); err != nil && os.IsNotExist(err) {
		log.Printf("Creating new prow yaml at path %s", fullOutputConfig)
		cmd := exec.Command("/usr/bin/config-forker", "--job-config", fullJobConfig, "--version", version, "--output", fullOutputConfig)
		bytes, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("ERROR: config-forker command output: %s : %s ", string(bytes), err)
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

		if !r.dryRun {
			_, err = gitCommand("-C", r.infraDir, "push", r.infraUrl, gitbranch)
			if err != nil {
				return err
			}
		}
	} else if err != nil {
		return err
	}

	if !r.dryRun {
		// Example at...
		// https://github.com/kubevirt/kubevirt/blob/main/hack/autobump-kubevirtci.sh
		// This should be idempotent, so it's okay if we call this multiple times
		log.Printf("Creating PR for new prow yamls")
		cmd := exec.Command("/usr/bin/pr-creator",
			"--org", "kubevirt",
			"--repo", "project-infra",
			"--branch", "main",
			"--github-token-path", r.githubTokenPath,
			"--title", fmt.Sprintf("Release configs for %s/%s release branch %s", r.org, r.repo, r.newBranch),
			"--body", "adds new release configs",
			"--source", fmt.Sprintf("kubevirt:%s", gitbranch),
			"--confirm",
		)
		bytes, err := cmd.CombinedOutput()
		if err != nil && !strings.Contains(string(bytes), "A pull request already exists") {
			log.Printf("ERROR: pr-creator command output: %s : %s ", string(bytes), err)
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

func (r *releaseData) verifyBranch() error {

	branches, err := r.getBranches()
	if err != nil {
		return err
	}

	for _, b := range branches {
		if b.Name != nil && *b.Name == r.newBranch {
			log.Printf("Release branch [%s] already exists", r.newBranch)
			return nil
		}
	}

	// branches are expected to be formatted as "release-x.y"
	re := regexp.MustCompile(`^release-\d*\.\d*$`)
	match := re.FindString(r.newBranch)
	if match != r.newBranch {
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

	branches, _, err := r.githubClient.Repositories.ListBranches(context.Background(), r.org, r.repo, &github.BranchListOptions{
		ListOptions: github.ListOptions{
			PerPage: 10000,
		},
	})
	if err != nil {
		return nil, err
	}
	r.allBranches = branches

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

func (r *releaseData) autoDetectData(autoReleaseCadance string, autoPromoteAfterDays int) error {

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

func (r *releaseData) verifyTag() error {
	// must be a valid semver version
	tagSemver, err := semver.NewVersion(r.tag)
	if err != nil {
		return err
	}

	expectedBranch := fmt.Sprintf("release-%d.%d", tagSemver.Major(), tagSemver.Minor())

	releases, err := r.getReleases()
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
		re := regexp.MustCompile(`^v\d*\.\d*.\d*-rc.\d*$`)
		match := re.FindString(r.tag)
		if match == "" && tagSemver.Patch() == 0 && !r.force {
			return fmt.Errorf("The tag [%s] must be promoted from a release candidate since it is the first release of a patch series.", r.tag)
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
	return nil
}

func (r *releaseData) cutNewTag() error {

	// checkout remote branch
	err := r.checkoutUpstream()
	if err != nil {
		return err
	}

	r.makeTag(r.tagBranch)

	return nil
}

func (r *releaseData) printData() {

	if r.dryRun {
		log.Print("DRY-RUN")
	}

	log.Print("Input Data")

	log.Printf("\trepoUrl: %s", r.repoUrl)
	log.Printf("\tnewTag: %s", r.tag)
	log.Printf("\tnewBranch: %s", r.newBranch)
	log.Printf("\torg: %s", r.org)
	log.Printf("\trepo: %s", r.repo)
	log.Printf("\tforce: %t", r.force)
	log.Printf("\tgithubTokenFile: %s", r.githubTokenPath)
}

func main() {
	newBranch := flag.String("new-branch", "", "New branch to cut from main.")
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

	tokenBytes, err := ioutil.ReadFile(*githubTokenFile)
	if err != nil {
		log.Fatalf("ERROR accessing github token: %s ", err)
	}
	token := strings.TrimSpace(string(tokenBytes))

	repoUrl := fmt.Sprintf("https://%s@github.com/%s/%s.git", token, *org, *repo)
	infraUrl := fmt.Sprintf("https://%s@github.com/kubevirt/project-infra.git", token)
	repoDir := fmt.Sprintf("%s/%s/https-%s", *cacheDir, *org, *repo)
	infraDir := fmt.Sprintf("%s/%s/https-%s", *cacheDir, "kubevirt", "project-infra")

	if *cleanCacheDir {
		os.RemoveAll(repoDir)
		os.RemoveAll(infraDir)
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
		tag:              *releaseTag,
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
		r.autoDetectData(*autoReleaseCadance, *autoPromoteAfterDays)
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

		err = r.cutNewBranch(*skipProw)
		if err != nil {
			log.Fatalf("ERROR Creating Branch: %s ", err)
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

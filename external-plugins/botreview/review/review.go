/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright the KubeVirt authors.
 *
 */

package review

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/go-diff/diff"
	gitv2 "sigs.k8s.io/prow/pkg/git/v2"
	"sigs.k8s.io/prow/pkg/github"
)

type KindOfChange interface {
	AddIfRelevant(fileDiff *diff.FileDiff)
	Review() BotReviewResult
	IsRelevant() bool
	MatchSubject(subject string) bool
}

func newPossibleReviewTypes() []KindOfChange {
	return []KindOfChange{
		&ProwJobImageUpdate{},
		&BumpKubevirtCI{},
		&ProwAutobump{},
		&KubeVirtUploader{},
	}
}

func GuessReviewTypes(fileDiffs []*diff.FileDiff) []KindOfChange {
	possibleReviewTypes := newPossibleReviewTypes()
	for _, fileDiff := range fileDiffs {
		for _, kindOfChange := range possibleReviewTypes {
			kindOfChange.AddIfRelevant(fileDiff)
		}
	}
	var result []KindOfChange
	for _, t := range possibleReviewTypes {
		if t.IsRelevant() {
			result = append(result, t)
		}
	}
	return result
}

type Reviewer struct {
	l       *logrus.Entry
	org     string
	repo    string
	num     int
	user    string
	action  github.PullRequestEventAction
	title   string
	dryRun  bool
	BaseSHA string
}

func NewReviewer(l *logrus.Entry, action github.PullRequestEventAction, org string, repo string, num int, user string, title string, dryRun bool) *Reviewer {
	return &Reviewer{
		l:      l,
		org:    org,
		repo:   repo,
		num:    num,
		user:   user,
		action: action,
		title:  title,
		dryRun: dryRun,
	}
}

func (r *Reviewer) withFields() *logrus.Entry {
	return r.l.WithField("dryRun", r.dryRun).WithField("org", r.org).WithField("repo", r.repo).WithField("pr", r.num).WithField("user", r.user).WithField("title", r.title)
}
func (r *Reviewer) info(message string) {
	r.withFields().Info(message)
}

func (r *Reviewer) fatalF(message string, args ...interface{}) {
	r.withFields().Fatalf(message, args...)
}

func (r *Reviewer) ReviewLocalCode() ([]BotReviewResult, error) {

	r.info("preparing review")

	diffCommand := exec.Command("git", "diff", "..main")
	if r.BaseSHA != "" {
		diffCommand = exec.Command("git", "diff", fmt.Sprintf("%s..%s", r.BaseSHA, "HEAD"))
	}
	output, err := diffCommand.Output()
	if err != nil {
		r.fatalF("could not fetch diff output: %v", err)
	}

	multiFileDiffReader := diff.NewMultiFileDiffReader(strings.NewReader(string(output)))
	files, err := multiFileDiffReader.ReadAllFiles()
	if err != nil {
		r.fatalF("could not create diffs from output: %v", err)
	}

	types := GuessReviewTypes(files)
	if len(types) == 0 {
		r.info("this PR didn't match any review type")
		return nil, nil
	}

	matchingTypes := []KindOfChange{}
	for _, t := range types {
		if t.MatchSubject(r.title) {
			matchingTypes = append(matchingTypes, t)
		}
	}

	if len(matchingTypes) == 0 {
		r.info("this PR title didn't match any expected titles")
		return nil, nil
	}

	results := []BotReviewResult{}
	for _, reviewType := range matchingTypes {
		result := reviewType.Review()
		results = append(results, result)
	}

	return results, nil
}

const botReviewCommentPattern = `@%s's review-bot says:

%s

%s

%s

**Note: botreview (kubevirt/project-infra#3100) is a Work In Progress!**
`
const holdPRComment = `Holding this PR because:
%s

/hold`
const canMergePRComment = "This PR does not require further manual action."

const approvePRComment = `This PR satisfies all automated review criteria.

/lgtm
/approve`
const unapprovePRComment = "This PR does not satisfy at least one automated review criteria."

type gitHubReviewClient interface {
	CreateComment(org, repo string, number int, comment string) error
	BotUser() (*github.UserData, error)
}

func (r *Reviewer) AttachReviewComments(botReviewResults []BotReviewResult, githubClient gitHubReviewClient) error {
	botUser, err := githubClient.BotUser()
	if err != nil {
		return fmt.Errorf("error while fetching user data: %v", err)
	}
	shouldNotMergeReasons := []string{}
	isApproved := true
	botReviewComments := make([]string, 0, len(botReviewResults))
	shortBotReviewComments := make([]string, 0, len(botReviewResults))
	for _, reviewResult := range botReviewResults {
		isApproved = isApproved && reviewResult.IsApproved()
		if reviewResult.ShouldNotMergeReason() != "" {
			shouldNotMergeReasons = append(shouldNotMergeReasons, reviewResult.ShouldNotMergeReason())
		}
		botReviewComments = append(botReviewComments, reviewResult.String())
		shortBotReviewComments = append(shortBotReviewComments, reviewResult.ShortString())
	}
	approveLabels := unapprovePRComment
	if isApproved {
		approveLabels = approvePRComment
	}
	holdComment := canMergePRComment
	if len(shouldNotMergeReasons) > 0 {
		holdComment = fmt.Sprintf(holdPRComment, newBulletList(shouldNotMergeReasons))
	}
	botReviewComment := fmt.Sprintf(
		botReviewCommentPattern,
		botUser.Login,
		strings.Join(botReviewComments, "\n"),
		approveLabels,
		holdComment,
	)
	if len(botReviewComment) > 2<<15 {
		botReviewComment = fmt.Sprintf(
			botReviewCommentPattern,
			botUser.Login,
			newBulletList(shortBotReviewComments),
			approveLabels,
			holdComment,
		)
	}
	if !r.dryRun {
		err = githubClient.CreateComment(r.org, r.repo, r.num, botReviewComment)
		if err != nil {
			return fmt.Errorf("error while creating review comment: %v", err)
		}
	} else {
		r.l.Info(fmt.Sprintf("dry-run: %s/%s#%d <- %s", r.org, r.repo, r.num, botReviewComment))
	}
	return nil
}

func newBulletList(shouldNotMergeReasons []string) string {
	return "* " + strings.Join(shouldNotMergeReasons, "\n* ")
}

type PRReviewOptions struct {
	PullRequestNumber int
	Org               string
	Repo              string
}

func PreparePullRequestReview(gitClient gitv2.ClientFactory, prReviewOptions PRReviewOptions, githubClient github.Client) (*github.PullRequest, string, error) {
	// checkout repo to a temporary directory to have it reviewed
	clone, err := gitClient.ClientFor(prReviewOptions.Org, prReviewOptions.Repo)
	if err != nil {
		logrus.WithError(err).Fatal("error cloning repo")
	}

	// checkout PR head commit, change dir
	pullRequest, err := githubClient.GetPullRequest(prReviewOptions.Org, prReviewOptions.Repo, prReviewOptions.PullRequestNumber)
	if err != nil {
		logrus.WithError(err).Fatal("error fetching PR")
	}
	err = clone.Checkout(pullRequest.Head.SHA)
	if err != nil {
		logrus.WithError(err).Fatal("error checking out PR head commit")
	}
	return pullRequest, clone.Directory(), err
}

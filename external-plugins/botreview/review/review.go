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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package review

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/go-diff/diff"
	"k8s.io/test-infra/prow/github"
	"os/exec"
	"strings"
)

type KindOfChange interface {
	AddIfRelevant(fileDiff *diff.FileDiff)
	Review() BotReviewResult
	IsRelevant() bool
}

type BotReviewResult interface {
	String() string
}

func newPossibleReviewTypes() []KindOfChange {
	return []KindOfChange{
		&ProwJobImageUpdate{},
		&BumpKubevirtCI{},
		&ProwAutobump{},
	}
}

func GuessReviewTypes(fileDiffs []*diff.FileDiff) []KindOfChange {
	possibleReviewTypes := newPossibleReviewTypes()
	for _, fileDiff := range fileDiffs {
		for _, kindOfChange := range possibleReviewTypes {
			kindOfChange.AddIfRelevant(fileDiff)
		}
	}
	result := []KindOfChange{}
	for _, t := range possibleReviewTypes {
		if t.IsRelevant() {
			result = append(result, t)
		}
	}
	return result
}

type BasicResult struct {
	message string
}

func (n BasicResult) String() string {
	return n.message
}

type Reviewer struct {
	l      *logrus.Entry
	org    string
	repo   string
	num    int
	user   string
	action github.PullRequestEventAction
	dryRun bool
}

func NewReviewer(l *logrus.Entry, action github.PullRequestEventAction, org string, repo string, num int, user string, dryRun bool) *Reviewer {
	return &Reviewer{
		l:      l,
		org:    org,
		repo:   repo,
		num:    num,
		user:   user,
		action: action,
		dryRun: dryRun,
	}
}

func (r *Reviewer) withFields() *logrus.Entry {
	return r.l.WithField("dryRun", r.dryRun).WithField("org", r.org).WithField("repo", r.repo).WithField("pr", r.num).WithField("user", r.user)
}
func (r *Reviewer) info(message string) {
	r.withFields().Info(message)
}
func (r *Reviewer) infoF(message string, args ...interface{}) {
	r.withFields().Infof(message, args...)
}
func (r *Reviewer) fatalF(message string, args ...interface{}) {
	r.withFields().Fatalf(message, args...)
}
func (r *Reviewer) debugF(message string, args ...interface{}) {
	r.withFields().Debugf(message, args...)
}

func (r *Reviewer) ReviewLocalCode() ([]BotReviewResult, error) {

	r.info("preparing review")

	diffCommand := exec.Command("git", "diff", "..main")
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
	if len(types) > 1 {
		r.info("doesn't look like a simple review, skipping")
		r.debugF("reviewTypes: %v", types)
		return nil, nil
	}

	results := []BotReviewResult{}
	for _, reviewType := range types {
		result := reviewType.Review()
		results = append(results, result)
	}

	return results, nil
}

func (r *Reviewer) AttachReviewComments(botReviewResults []BotReviewResult, githubClient github.Client) error {
	botUser, err := githubClient.BotUser()
	if err != nil {
		return fmt.Errorf("error while fetching user data: %v", err)
	}
	for _, reviewResult := range botReviewResults {
		botReviewComment := fmt.Sprintf("@%s's review-bot says:\n\n%v", botUser.Login, reviewResult)
		if !r.dryRun {
			err = githubClient.CreateComment(r.org, r.repo, r.num, botReviewComment)
			if err != nil {
				return fmt.Errorf("error while creating review comment: %v", err)
			}
		} else {
			r.l.Info(fmt.Sprintf("dry-run: %s/%s#%d <- %s", r.org, r.repo, r.num, botReviewComment))
		}
	}
	return nil
}

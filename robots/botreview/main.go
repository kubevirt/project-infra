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

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/pkg/flagutil"
	"kubevirt.io/project-infra/external-plugins/botreview/review"
	"sigs.k8s.io/prow/pkg/config/secret"
	prowflagutil "sigs.k8s.io/prow/pkg/flagutil"
	"sigs.k8s.io/prow/pkg/github"
)

const robotName = "botreview"

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.DebugLevel)
}

type options struct {
	review.PRReviewOptions

	dryRun bool
	github prowflagutil.GitHubOptions
}

func (o *options) Validate() error {
	for idx, group := range []flagutil.OptionGroup{&o.github} {
		if err := group.Validate(o.dryRun); err != nil {
			return fmt.Errorf("%d: %w", idx, err)
		}
	}

	if o.Org == "" || o.Repo == "" || o.PullRequestNumber == 0 {
		return fmt.Errorf("org, repo and pr-number are required")
	}

	return nil
}

func gatherOptions() options {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.BoolVar(&o.dryRun, "dry-run", true, "Dry run for testing. Uses API tokens but does not mutate.")
	fs.StringVar(&o.Org, "org", "kubevirt", "Pull request github org.")
	fs.StringVar(&o.Repo, "repo", "", "Pull request github repo.")
	fs.IntVar(&o.PullRequestNumber, "pr-number", 0, "Pull request to review.")
	for _, group := range []flagutil.OptionGroup{&o.github} {
		group.AddFlags(fs)
	}
	fs.Parse(os.Args[1:])
	return o
}

func main() {
	o := gatherOptions()
	if err := o.Validate(); err != nil {
		logrus.Fatalf("Invalid options: %v", err)
	}

	log := logrus.StandardLogger().WithField("robot", robotName)

	if err := secret.Add(o.github.TokenPath); err != nil {
		logrus.WithError(err).Fatal("error starting secrets agent")
	}

	githubClient, err := o.github.GitHubClientWithAccessToken(string(secret.GetSecret(o.github.TokenPath)))
	if err != nil {
		logrus.WithError(err).Fatal("error getting GitHub client")
	}
	cacheDir := ""
	gitClient, err := o.github.GitClientFactory("", &cacheDir, o.dryRun, false)
	if err != nil {
		logrus.WithError(err).Fatal("error getting Git client")
	}

	prReviewOptions := review.PRReviewOptions{
		PullRequestNumber: o.PullRequestNumber,
		Org:               o.Org,
		Repo:              o.Repo,
	}
	pullRequest, cloneDirectory, err := review.PreparePullRequestReview(gitClient, prReviewOptions, githubClient)
	if err != nil {
		logrus.WithError(err).Fatal("error preparing pull request for review")
	}
	err = os.Chdir(cloneDirectory)
	if err != nil {
		logrus.WithError(err).Fatal("error changing to directory")
	}

	// Perform review
	user, err := githubClient.BotUser()
	if err != nil {
		logrus.WithError(err).Fatal("error getting bot user")
	}
	reviewer := review.NewReviewer(log, github.PullRequestActionEdited, o.Org, o.Repo, o.PullRequestNumber, user.Login, pullRequest.Title, o.dryRun)
	reviewer.BaseSHA = pullRequest.Base.SHA
	botReviewResults, err := reviewer.ReviewLocalCode()
	if err != nil {
		log.Info("no review results, cancelling review comments")
	}
	if len(botReviewResults) == 0 {
		return
	}

	err = reviewer.AttachReviewComments(botReviewResults, githubClient)
	if err != nil {
		log.Errorf("error while attaching review comments: %v", err)
	}
}

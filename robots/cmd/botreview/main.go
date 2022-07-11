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
	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/pkg/flagutil"
	"k8s.io/test-infra/prow/config/secret"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/github"
	"kubevirt.io/project-infra/external-plugins/botreview/review"
	"os"
)

const robotName = "botreview"

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.DebugLevel)
}

type options struct {
	pullRequestNumber int
	org               string
	repo              string

	dryRun bool
	github prowflagutil.GitHubOptions
	labels prowflagutil.Strings
}

func (o *options) Validate() error {
	for idx, group := range []flagutil.OptionGroup{&o.github} {
		if err := group.Validate(o.dryRun); err != nil {
			return fmt.Errorf("%d: %w", idx, err)
		}
	}

	if o.org == "" || o.repo == "" || o.pullRequestNumber == 0 {
		return fmt.Errorf("org, repo and pr-number are required")
	}

	return nil
}

func gatherOptions() options {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.BoolVar(&o.dryRun, "dry-run", true, "Dry run for testing. Uses API tokens but does not mutate.")
	fs.StringVar(&o.org, "org", "kubevirt", "Pull request github org.")
	fs.StringVar(&o.repo, "repo", "", "Pull request github repo.")
	fs.IntVar(&o.pullRequestNumber, "pr-number", 0, "Pull request to review.")
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

	githubClient := o.github.GitHubClientWithAccessToken(string(secret.GetSecret(o.github.TokenPath)))
	gitClient, err := o.github.GitClient(o.dryRun)
	if err != nil {
		logrus.WithError(err).Fatal("error getting Git client")
	}
	user, err := githubClient.BotUser()
	if err != nil {
		logrus.WithError(err).Fatal("error getting bot user")
	}

	// checkout repo to a temporary directory to have it reviewed
	clone, err := gitClient.Clone(o.org, o.repo)
	if err != nil {
		logrus.WithError(err).Fatal("error cloning repo")
	}

	// checkout PR head commit, change dir
	pullRequest, err := githubClient.GetPullRequest(o.org, o.repo, o.pullRequestNumber)
	if err != nil {
		logrus.WithError(err).Fatal("error fetching PR")
	}
	err = clone.Checkout(pullRequest.Head.SHA)
	if err != nil {
		logrus.WithError(err).Fatal("error checking out PR head commit")
	}
	err = os.Chdir(clone.Directory())
	if err != nil {
		logrus.WithError(err).Fatal("error changing to directory")
	}

	// Perform review
	reviewer := review.NewReviewer(log, github.PullRequestActionEdited, o.org, o.repo, o.pullRequestNumber, user.Login, o.dryRun)
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

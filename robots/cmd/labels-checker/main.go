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
 * Copyright 2021 Red Hat, Inc.
 *
 */
package main

import (
	"context"
	"flag"
	"fmt"
	"k8s.io/test-infra/prow/config/secret"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type options struct {
	tokenPath           string
	endpoint            string
	org                 string
	repo                string
	author              string
	branchName          string
	ensureLabelsMissing string
}

func (o *options) validate() error {
	if o.org == "" {
		return fmt.Errorf("org is required")
	}
	if o.repo == "" {
		return fmt.Errorf("repo is required")
	}
	if o.author == "" {
		return fmt.Errorf("author is required")
	}
	if o.branchName == "" {
		return fmt.Errorf("branch-name is required")
	}
	return nil
}

func (o *options) getEnsureLabelsMissing() []string {
	return strings.Split(o.ensureLabelsMissing, ",")
}

var o = options{}

func init() {
	flag.StringVar(&o.tokenPath, "github-token-path", "/etc/github/oauth", "Path to the file containing the GitHub OAuth secret.")
	flag.StringVar(&o.endpoint, "github-endpoint", "https://api.github.com/", "GitHub's API endpoint (may differ for enterprise).")
	flag.StringVar(&o.org, "org", "", "The org for the PR.")
	flag.StringVar(&o.repo, "repo", "", "The repo for the PR.")
	flag.StringVar(&o.author, "author", "", "The author for the PR.")
	flag.StringVar(&o.branchName, "branch-name", "", "The branch name for the PR.")
	flag.StringVar(&o.ensureLabelsMissing, "ensure-labels-missing", "lgtm", "What labels have to be missing on the PR (list of comma separated labels).")
}

func main() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	// TODO: Use global option from the prow config.
	logrus.SetLevel(logrus.DebugLevel)

	flag.Parse()
	if err := o.validate(); err != nil {
		log().WithError(err).Fatal("Invalid arguments provided.")
	}

	ctx := context.Background()
	var client *github.Client
	if o.tokenPath == "" {
		var err error
		client, err = github.NewEnterpriseClient(o.endpoint, o.endpoint, nil)
		if err != nil {
			log().Panicln(err)
		}
	} else {
		err := secret.Add(o.tokenPath)
		if err != nil {
			log().Fatalf("Failed to load token from path %s: %v", o.tokenPath, err)
		}
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: string(secret.GetSecret(o.tokenPath))},
		)
		client, err = github.NewEnterpriseClient(o.endpoint, o.endpoint, oauth2.NewClient(ctx, ts))
		if err != nil {
			log().Panicln(err)
		}
	}

	prs, _, err := client.PullRequests.List(ctx, o.org, o.repo, &github.PullRequestListOptions{
		State:       "open",
		Head:        fmt.Sprintf("%s:%s", o.author, o.branchName),
		ListOptions: github.ListOptions{},
	})
	if err != nil {
		log().WithError(err).Fatal("failed to find PR")
	} else if len(prs) == 0 {
		log().Info("No PR found")
		os.Exit(0)
	} else if len(prs) > 1 {
		log().Fatalf("More than one PR found: %+v", prs)
	}

	if checkAnyLabelExists(prs[0], o.getEnsureLabelsMissing()) {
		log().WithField("PR", prs[0].GetNumber()).Fatalf("ensureLabelsMissing: some labels were present that shouldn't be")
	}

}

func checkAnyLabelExists(prToCheck *github.PullRequest, labelsToCheck []string) bool {
	labels := map[string]struct{}{}
	for _, label := range prToCheck.Labels {
		name := *label.Name
		labels[name] = struct{}{}
	}
	labelsExist := false
	for _, label := range labelsToCheck {
		if _, exists := labels[label]; exists {
			log().WithField("PR", prToCheck.GetNumber()).Infof("label %s exists", label)
			labelsExist = true
		}
	}
	return labelsExist
}

func log() *logrus.Entry {
	return logrus.StandardLogger().WithField("robot", "labels-checker")
}

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
 * Copyright the KubeVirt Authors.
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"k8s.io/test-infra/pkg/flagutil"
	"kubevirt.io/project-infra/external-plugins/referee/ghgraphql"
	"kubevirt.io/project-infra/external-plugins/referee/metrics"
	"kubevirt.io/project-infra/external-plugins/referee/server"
	"sigs.k8s.io/prow/pkg/config/secret"
	prowflagutil "sigs.k8s.io/prow/pkg/flagutil"
	"sigs.k8s.io/prow/pkg/interrupts"
	"sigs.k8s.io/prow/pkg/pluginhelp/externalplugins"
)

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.DebugLevel)
}

type options struct {
	port int

	dryRun                               bool
	maximumNumberOfAllowedRetestComments int

	github                    prowflagutil.GitHubOptions
	webhookSecretFile         string
	team                      string
	initialRetestRepositories string
}

var retestRepoOptionRegex = regexp.MustCompile(`^[^\s/,]+/[^\s/,]+(,[^\s/,]+/[^\s/,]+)*$`)

type RepoIdentifier struct {
	Org  string
	Repo string
}

func (o *options) InitialRetestRepositories() []RepoIdentifier {
	if !retestRepoOptionRegex.MatchString(o.initialRetestRepositories) {
		return nil
	}
	repoIds := strings.Split(o.initialRetestRepositories, ",")
	result := make([]RepoIdentifier, 0, len(repoIds))
	for _, repoId := range repoIds {
		split := strings.Split(repoId, "/")
		result = append(result, RepoIdentifier{Org: split[0], Repo: split[1]})
	}
	return result
}
func (o *options) Validate() error {
	for idx, group := range []flagutil.OptionGroup{&o.github} {
		if err := group.Validate(o.dryRun); err != nil {
			return fmt.Errorf("%d: %w", idx, err)
		}
	}
	if o.initialRetestRepositories != "" {
		if !retestRepoOptionRegex.MatchString(o.initialRetestRepositories) {
			return fmt.Errorf("%q doesn't match org/repo1,org/repo2,... ", o.initialRetestRepositories)
		}
	}

	return nil
}

func gatherOptions() options {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.IntVar(&o.port, "port", 8888, "Port to listen on.")
	fs.BoolVar(&o.dryRun, "dry-run", true, "Dry run for testing. Uses API tokens but does not mutate.")
	fs.StringVar(&o.webhookSecretFile, "hmac-secret-file", "/etc/webhook/hmac", "Path to the file containing the GitHub HMAC secret.")
	fs.StringVar(&o.team, "team", "sig-buildsystem", "Name of the GitHub team that should be pinged.")
	fs.StringVar(&o.initialRetestRepositories, "initial-retest-repositories", "kubevirt/kubevirt", "Comma-separated names of GitHub repositories to fetch the number of retest comments for open lgtm/approved pull request in format org/repo1,org/repo2,... ")
	fs.IntVar(&o.maximumNumberOfAllowedRetestComments, "max-no-of-allowed-retest-comments", server.DefaultMaximumNumberOfAllowedRetestComments, "Maximum number of allowed retest comments.")
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

	log := logrus.StandardLogger().WithField("plugin", server.PluginName)

	if err := secret.Add(o.github.TokenPath, o.webhookSecretFile); err != nil {
		logrus.WithError(err).Fatal("Error starting secrets agent.")
	}

	githubClient, err := o.github.GitHubClientWithAccessToken(string(secret.GetSecret(o.github.TokenPath)))
	if err != nil {
		logrus.WithError(err).Fatal("error getting github client")
	}

	botUserData, err := githubClient.BotUser()
	if err != nil {
		logrus.WithError(err).Fatal("Error getting bot name.")
	}

	token, err := os.ReadFile(o.github.TokenPath)
	if err != nil {
		logrus.Fatalf("failed to use github token path %s: %v", o.github.TokenPath, err)
	}
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: strings.TrimSpace(string(token))},
	)
	httpClient := oauth2.NewClient(context.Background(), src)

	gitHubGQLClient := ghgraphql.NewClient(githubv4.NewClient(httpClient))

	go initializeRetestsForRepositories(log, o.InitialRetestRepositories(), gitHubGQLClient)

	pluginServer := &server.Server{
		TokenGenerator: secret.GetTokenGenerator(o.webhookSecretFile),
		BotName:        botUserData.Name,
		Team:           o.team,

		GithubClient:    githubClient,
		GHGraphQLClient: gitHubGQLClient,
		Log:             log,

		DryRun:                               o.dryRun,
		MaximumNumberOfAllowedRetestComments: o.maximumNumberOfAllowedRetestComments,
	}

	mux := http.NewServeMux()
	mux.Handle("/", pluginServer)
	externalplugins.ServeExternalPluginHelp(mux, log, server.HelpProvider)
	httpServer := &http.Server{Addr: ":" + strconv.Itoa(o.port), Handler: mux}
	metrics.AddMetricsHandler(mux)
	defer interrupts.WaitForGracefulShutdown()
	interrupts.ListenAndServe(httpServer, 5*time.Second)

}

func initializeRetestsForRepositories(log *logrus.Entry, repoIds []RepoIdentifier, gqlClient ghgraphql.GitHubGraphQLClient) {
	for _, repoId := range repoIds {
		org, repo := repoId.Org, repoId.Repo
		pullRequests, err := gqlClient.FetchOpenApprovedAndLGTMedPRs(org, repo)
		if err != nil {
			log.Fatalf("failed to fetch pull requests for %s/%s: %v", org, repo, err)
		}

		log.Infof("checking %d PRs for retests", len(pullRequests.PRs))
		for _, pr := range pullRequests.PRs {
			prLog := log.WithField("pr_url", fmt.Sprintf("https://github.com/%s/%s/pull/%d", org, repo, pr.Number)).WithField("pr_title", pr.Title)
			prTimeLineForLastCommit, err := gqlClient.FetchPRTimeLineForLastCommit(org, repo, pr.Number)
			if err != nil {
				prLog.Fatalf("failed to fetch number of retest comments for pr %s/%s#%d: %v", org, repo, pr.Number, err)
			}
			prLog.Infof("got number of retests")
			metrics.SetForPullRequest(org, repo, pr.Number, prTimeLineForLastCommit.NumberOfRetestComments)
		}
	}
}

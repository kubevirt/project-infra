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
	"github.com/shurcooL/githubv4"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"k8s.io/test-infra/pkg/flagutil"
	"k8s.io/test-infra/prow/config/secret"
	prowflagutil "k8s.io/test-infra/prow/flagutil"
	"k8s.io/test-infra/prow/interrupts"
	"k8s.io/test-infra/prow/pluginhelp/externalplugins"
	"kubevirt.io/project-infra/external-plugins/referee/ghgraphql"
	"kubevirt.io/project-infra/external-plugins/referee/server"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.DebugLevel)
}

type options struct {
	port int

	dryRun bool
	github prowflagutil.GitHubOptions
	labels prowflagutil.Strings

	webhookSecretFile string
	team              string
}

func (o *options) Validate() error {
	for idx, group := range []flagutil.OptionGroup{&o.github} {
		if err := group.Validate(o.dryRun); err != nil {
			return fmt.Errorf("%d: %w", idx, err)
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

	githubClient := o.github.GitHubClientWithAccessToken(string(secret.GetSecret(o.github.TokenPath)))

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

	gitHubGQLCLient := ghgraphql.NewClient(githubv4.NewClient(httpClient))

	pluginServer := &server.Server{
		TokenGenerator: secret.GetTokenGenerator(o.webhookSecretFile),
		BotName:        botUserData.Name,
		Team:           o.team,

		GithubClient:    githubClient,
		GHGraphQLClient: gitHubGQLCLient,
		Log:             log,

		DryRun: o.dryRun,
	}

	mux := http.NewServeMux()
	mux.Handle("/", pluginServer)
	externalplugins.ServeExternalPluginHelp(mux, log, server.HelpProvider)
	httpServer := &http.Server{Addr: ":" + strconv.Itoa(o.port), Handler: mux}
	defer interrupts.WaitForGracefulShutdown()
	interrupts.ListenAndServe(httpServer, 5*time.Second)

}

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
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"k8s.io/test-infra/pkg/flagutil"
	"kubevirt.io/project-infra/external-plugins/botreview/server"
	"sigs.k8s.io/prow/pkg/config/secret"
	prowflagutil "sigs.k8s.io/prow/pkg/flagutil"
	"sigs.k8s.io/prow/pkg/interrupts"
	"sigs.k8s.io/prow/pkg/pluginhelp/externalplugins"
)

const pluginName = "botreview"

func init() {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.DebugLevel)
}

type options struct {
	port int

	dryRun bool
	github prowflagutil.GitHubOptions

	webhookSecretFile string
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

	log := logrus.StandardLogger().WithField("plugin", pluginName)

	if err := secret.Add(o.github.TokenPath, o.webhookSecretFile); err != nil {
		logrus.WithError(err).Fatal("Error starting secrets agent.")
	}

	githubClient, err := o.github.GitHubClientWithAccessToken(string(secret.GetSecret(o.github.TokenPath)))
	if err != nil {
		logrus.WithError(err).Fatal("Error getting GitHub client.")
	}
	// TODO: check setup ok
	cacheDir := ""
	gitClient, err := o.github.GitClientFactory("", &cacheDir, o.dryRun, false)
	if err != nil {
		logrus.WithError(err).Fatal("Error getting Git client.")
	}
	interrupts.OnInterrupt(func() {
		if err := gitClient.Clean(); err != nil {
			logrus.WithError(err).Error("Could not clean up git client cache.")
		}
	})

	botUserData, err := githubClient.BotUser()
	if err != nil {
		logrus.WithError(err).Fatal("Error getting bot name.")
	}

	pluginServer := &server.Server{
		TokenGenerator: secret.GetTokenGenerator(o.webhookSecretFile),
		BotName:        botUserData.Name,

		GitClient: gitClient,
		Ghc:       githubClient,
		Log:       log,

		DryRun: o.dryRun,
	}

	mux := http.NewServeMux()
	mux.Handle("/", pluginServer)
	externalplugins.ServeExternalPluginHelp(mux, log, server.HelpProvider)
	httpServer := &http.Server{Addr: ":" + strconv.Itoa(o.port), Handler: mux}
	defer interrupts.WaitForGracefulShutdown()
	interrupts.ListenAndServe(httpServer, 5*time.Second)

}

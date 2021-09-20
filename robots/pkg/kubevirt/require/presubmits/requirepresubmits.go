/*
Copyright 2021 The KubeVirt Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package presubmits

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"

	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	"k8s.io/test-infra/prow/config"
	"sigs.k8s.io/yaml"

	"kubevirt.io/project-infra/robots/pkg/querier"
)

const orgAndRepoForJobConfig = "kubevirt/kubevirt"

type options struct {
	jobConfigPathKubevirtPresubmits string
}

var requirePresubmitsCommand = &cobra.Command{
	Use: "presubmits",
	Short: "kubevirt require presubmits requires presubmit job definitions in project-infra for kubevirt/kubevirt repo",
	Run: func(cmd *cobra.Command, args []string) {

		err := cmd.InheritedFlags().Parse(args)
		if err != nil {
			fmt.Println(fmt.Errorf("failed to parse args: %v", err))
			os.Exit(1)
		}

		if err := o.Validate(); err != nil {
			log().WithError(err).Error("Invalid arguments provided.")
			os.Exit(1)
		}

		run(cmd)
	},
}

var o = options{}

func NewRequirePresubmitsCommand() *cobra.Command {
	return requirePresubmitsCommand
}

func init() {
	requirePresubmitsCommand.PersistentFlags().StringVar(&o.jobConfigPathKubevirtPresubmits, "job-config-path-kubevirt-presubmits", "", "The directory of the kubevirt presubmit job definitions")
}

func (o *options) Validate() error {
	if _, err := os.Stat(o.jobConfigPathKubevirtPresubmits); os.IsNotExist(err) {
		return fmt.Errorf("jobConfigPathKubevirtPresubmits is required: %v", err)
	}
	return nil
}

func run(cmd *cobra.Command) {

	logrus.SetFormatter(&logrus.JSONFormatter{})
	// TODO: Use global option from the prow config.
	logrus.SetLevel(logrus.DebugLevel)

	tokenPath, err := cmd.InheritedFlags().GetString("github-token-path")
	if err != nil {
		log().Panicln(err)
	}
	endPoint, err := cmd.InheritedFlags().GetString("github-endpoint")
	if err != nil {
		log().Panicln(err)
	}

	ctx := context.Background()
	var client *github.Client
	if tokenPath == "" {
		var err error
		client, err = github.NewEnterpriseClient(endPoint, endPoint, nil)
		if err != nil {
			log().Panicln(err)
		}
	} else {
		token, err := ioutil.ReadFile(tokenPath)
		if err != nil {
			log().Panicln(err)
		}
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: string(token)},
		)
		client, err = github.NewEnterpriseClient(endPoint, endPoint, oauth2.NewClient(ctx, ts))
		if err != nil {
			log().Panicln(err)
		}
	}

	jobConfig, err := config.ReadJobConfig(o.jobConfigPathKubevirtPresubmits)
	if err != nil {
		log().Panicln(err)
	}

	releases, _, err := client.Repositories.ListReleases(ctx, "kubernetes", "kubernetes", nil)
	if err != nil {
		log().Panicln(err)
	}
	releases = querier.ValidReleases(releases)
	if len(releases) == 0 {
		log().Info("No release found, nothing to do.")
		os.Exit(0)
	}

	latestReleaseSemver := querier.ParseRelease(releases[0])

	dryRun, err := cmd.InheritedFlags().GetBool("dry-run")
	if err != nil {
		log().Panicln(err)
	}

	updated := UpdatePresubmitsAlwaysRunAndOptionalFields(&jobConfig, latestReleaseSemver)
	if !updated && !dryRun {
		log().Info(fmt.Sprintf("presubmit jobs for %v weren't modified, nothing to do.", latestReleaseSemver))
		os.Exit(0)
	}

	marshalledConfig, err := yaml.Marshal(&jobConfig)
	if err != nil {
		log().WithError(err).Error("Failed to marshall jobconfig")
	}

	if dryRun {
		_, err = os.Stdout.Write(marshalledConfig)
		if err != nil {
			log().WithError(err).Error("Failed to write jobconfig")
		}
		os.Exit(0)
	}

	err = ioutil.WriteFile(o.jobConfigPathKubevirtPresubmits, marshalledConfig, os.ModePerm)
	if err != nil {
		log().WithError(err).Error("Failed to write jobconfig")
	}
}

func UpdatePresubmitsAlwaysRunAndOptionalFields(jobConfig *config.JobConfig, latestReleaseSemver *querier.SemVer) (updated bool) {
	jobsToCheck := map[string]string{
		createPresubmitJobName(latestReleaseSemver, "sig-network"): "",
		createPresubmitJobName(latestReleaseSemver, "sig-storage"): "",
		createPresubmitJobName(latestReleaseSemver, "sig-compute"): "",
		createPresubmitJobName(latestReleaseSemver, "operator"): "",
	}

	for index := range jobConfig.PresubmitsStatic[orgAndRepoForJobConfig] {
		job := &jobConfig.PresubmitsStatic[orgAndRepoForJobConfig][index]
		name := job.Name
		if _, exists := jobsToCheck[name]; !exists {
			continue
		}

		// phase 1: always_run: false -> true
		if !job.AlwaysRun {
			job.AlwaysRun = true
			updated = true

			// -- fix skip_report: true -> false
			job.SkipReport = false

			continue
		}

		// phase 2: optional: true -> false
		if job.Optional {
			job.Optional = false
			updated = true
		}

	}

	return
}

func createPresubmitJobName(latestReleaseSemver *querier.SemVer, sigName string) string {
	return fmt.Sprintf("pull-kubevirt-e2e-k8s-%s.%s-%s", latestReleaseSemver.Major, latestReleaseSemver.Minor, sigName)
}

func log() *logrus.Entry {
	return logrus.StandardLogger().WithField("robot", "kubevirt require presubmit")
}

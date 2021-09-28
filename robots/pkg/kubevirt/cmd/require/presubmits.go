/*
 * Copyright 2021 The KubeVirt Authors.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package require

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"k8s.io/test-infra/prow/config"
	"sigs.k8s.io/yaml"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/cmd/flags"
	kv_github "kubevirt.io/project-infra/robots/pkg/kubevirt/github"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/jobconfig"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/log"
	"kubevirt.io/project-infra/robots/pkg/querier"
)

type requirePresubmitsOptions struct {
	jobConfigPathKubevirtPresubmits string
}

func (o requirePresubmitsOptions) Validate() error {
	if _, err := os.Stat(o.jobConfigPathKubevirtPresubmits); os.IsNotExist(err) {
		return fmt.Errorf("jobConfigPathKubevirtPresubmits is required: %v", err)
	}
	return nil
}

const shortUsage = "kubevirt require presubmits moves presubmit job definitions for kubevirt to being required to merge"

var requirePresubmitsCommand = &cobra.Command{
	Use:   "presubmits",
	Short: shortUsage,
	Long: fmt.Sprintf(`%s

For each of the sigs (%s)
it aims to make the jobs for the latest kubevirtci provider
required and run on every PR. This is done in two stages.
First it sets for a job that doesn't always run the

	always_run: true
	optional: false

flag. This will make the job run on every PR but failing checks
will not block the merge.

On second stage, it removes the

	optional: false

which makes the job required to pass for merges to occur with tide.
`, shortUsage, strings.Join(jobconfig.SigNames, ", ")),
	Run: run,
}

var requirePresubmitsOpts = requirePresubmitsOptions{}

func RequirePresubmitsCommand() *cobra.Command {
	return requirePresubmitsCommand
}

func init() {
	requirePresubmitsCommand.PersistentFlags().StringVar(&requirePresubmitsOpts.jobConfigPathKubevirtPresubmits, "job-config-path-kubevirt-presubmits", "", "The directory of the kubevirt presubmit job definitions")
}

func run(cmd *cobra.Command, args []string) {
	flags.ParseFlagsOrExit(cmd, args, requirePresubmitsOpts)

	ctx := context.Background()
	client := kv_github.NewGitHubClient(ctx)

	jobConfig, err := config.ReadJobConfig(requirePresubmitsOpts.jobConfigPathKubevirtPresubmits)
	if err != nil {
		log.Log().Panicln(err)
	}

	releases, _, err := client.Repositories.ListReleases(ctx, "kubernetes", "kubernetes", nil)
	if err != nil {
		log.Log().Panicln(err)
	}
	releases = querier.ValidReleases(releases)
	if len(releases) == 0 {
		log.Log().Info("No release found, nothing to do.")
		os.Exit(0)
	}

	latestReleaseSemver := querier.ParseRelease(releases[0])

	updated := updatePresubmitsAlwaysRunAndOptionalFields(&jobConfig, latestReleaseSemver)
	if !updated && !flags.Options.DryRun {
		log.Log().Info(fmt.Sprintf("presubmit jobs for %v weren't modified, nothing to do.", latestReleaseSemver))
		os.Exit(0)
	}

	marshalledConfig, err := yaml.Marshal(&jobConfig)
	if err != nil {
		log.Log().WithError(err).Error("Failed to marshall jobconfig")
	}

	if flags.Options.DryRun {
		_, err = os.Stdout.Write(marshalledConfig)
		if err != nil {
			log.Log().WithError(err).Error("Failed to write jobconfig")
		}
		os.Exit(0)
	}

	err = ioutil.WriteFile(requirePresubmitsOpts.jobConfigPathKubevirtPresubmits, marshalledConfig, os.ModePerm)
	if err != nil {
		log.Log().WithError(err).Error("Failed to write jobconfig")
	}
}

func updatePresubmitsAlwaysRunAndOptionalFields(jobConfig *config.JobConfig, latestReleaseSemver *querier.SemVer) (updated bool) {
	jobsToCheck := map[string]string{}
	for _, sigName := range jobconfig.SigNames {
		jobsToCheck[jobconfig.CreatePresubmitJobName(latestReleaseSemver, sigName)] = ""
	}

	for index := range jobConfig.PresubmitsStatic[jobconfig.OrgAndRepoForJobConfig] {
		job := &jobConfig.PresubmitsStatic[jobconfig.OrgAndRepoForJobConfig][index]
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

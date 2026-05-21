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
	"os"
	"strconv"
	"strings"

	"kubevirt.io/project-infra/pkg/kubevirt/cmd/flags"
	kvgithub "kubevirt.io/project-infra/pkg/kubevirt/github"
	"kubevirt.io/project-infra/pkg/kubevirt/log"
	"kubevirt.io/project-infra/pkg/kubevirt/prowjobconfigs"
	"kubevirt.io/project-infra/pkg/querier"

	"github.com/spf13/cobra"

	"sigs.k8s.io/prow/pkg/config"
	"sigs.k8s.io/yaml"
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
`, shortUsage, strings.Join(prowjobconfigs.SigNames, ", ")),
	RunE: run,
}

var requirePresubmitsOpts = requirePresubmitsOptions{}

func RequirePresubmitsCommand() *cobra.Command {
	return requirePresubmitsCommand
}

func init() {
	requirePresubmitsCommand.PersistentFlags().StringVar(&requirePresubmitsOpts.jobConfigPathKubevirtPresubmits, "job-config-path-kubevirt-presubmits", "github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-presubmits.yaml", "The directory of the kubevirt presubmit job definitions")
}

func run(cmd *cobra.Command, args []string) error {
	err := flags.ParseFlags(cmd, args, requirePresubmitsOpts)
	if err != nil {
		return err
	}

	ctx := context.Background()
	client, err := kvgithub.NewGitHubClient(ctx)
	if err != nil {
		return err
	}

	jobConfig, err := config.ReadJobConfig(requirePresubmitsOpts.jobConfigPathKubevirtPresubmits)
	if err != nil {
		return fmt.Errorf("failed to read jobconfig %s: %v", requirePresubmitsOpts.jobConfigPathKubevirtPresubmits, err)
	}

	releases, _, err := client.Repositories.ListReleases(ctx, "kubernetes", "kubernetes", nil)
	if err != nil {
		return fmt.Errorf("failed to list releases: %v", err)
	}
	releases = querier.ValidReleases(releases)
	if len(releases) == 0 {
		log.Log().Info("No release found, nothing to do.")
		return nil
	}

	latestReleaseSemver := querier.ParseRelease(releases[0])

	updated, phase := updatePresubmitsAlwaysRunAndOptionalFields(&jobConfig, latestReleaseSemver)
	if !updated && !flags.Options.DryRun {
		log.Log().Info(fmt.Sprintf("presubmit jobs for %v weren't modified, nothing to do.", latestReleaseSemver))
		return nil
	}
	if phase == phase2 {
		previousReleaseSemver := querier.SemVer{Major: latestReleaseSemver.Major, Minor: strconv.Itoa(latestReleaseSemver.MinorInt() - 1)}
		setPresubmitsToRunBeforeMergeOnly(&jobConfig, &previousReleaseSemver)
	}

	marshalledConfig, err := yaml.Marshal(&jobConfig)
	if err != nil {
		return fmt.Errorf("failed to marshall jobconfig %s: %v", requirePresubmitsOpts.jobConfigPathKubevirtPresubmits, err)
	}

	if flags.Options.DryRun {
		_, err = os.Stdout.Write(marshalledConfig)
		if err != nil {
			return fmt.Errorf("failed to write jobconfig %s: %v", requirePresubmitsOpts.jobConfigPathKubevirtPresubmits, err)
		}
		return nil
	}

	err = os.WriteFile(requirePresubmitsOpts.jobConfigPathKubevirtPresubmits, marshalledConfig, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to write jobconfig %s: %v", requirePresubmitsOpts.jobConfigPathKubevirtPresubmits, err)
	}
	return nil
}

type phase int

const (
	phase1 = iota
	phase2
)

func updatePresubmitsAlwaysRunAndOptionalFields(jobConfig *config.JobConfig, releaseSemver *querier.SemVer) (updated bool, phase phase) {
	jobsToCheck := map[string]string{}
	for _, sigName := range prowjobconfigs.SigNames {
		jobsToCheck[prowjobconfigs.CreatePresubmitJobName(releaseSemver, sigName)] = ""
	}
	jobsToCheck[prowjobconfigs.CreatePresubmitJobName(releaseSemver, "sig-compute-serial")] = ""

	for index := range jobConfig.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig] {
		job := &jobConfig.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig][index]
		name := job.Name
		if _, exists := jobsToCheck[name]; !exists {
			continue
		}

		// phase 1: always_run: false -> true
		if !job.AlwaysRun {
			job.AlwaysRun = true
			updated = true
			phase = phase1

			// -- fix skip_report: true -> false
			job.SkipReport = false

			continue
		}

		// phase 2: optional: true -> false
		if job.Optional {
			job.Optional = false
			updated = true
			phase = phase2
		}

	}

	return
}

func setPresubmitsToRunBeforeMergeOnly(jobConfig *config.JobConfig, releaseSemver *querier.SemVer) {
	jobsToCheck := map[string]string{}
	for _, sigName := range prowjobconfigs.SigNames {
		jobsToCheck[prowjobconfigs.CreatePresubmitJobName(releaseSemver, sigName)] = ""
	}
	jobsToCheck[prowjobconfigs.CreatePresubmitJobName(releaseSemver, "sig-compute-serial")] = ""

	for index := range jobConfig.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig] {
		job := &jobConfig.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig][index]
		name := job.Name
		if _, exists := jobsToCheck[name]; !exists {
			continue
		}

		if job.AlwaysRun {
			job.AlwaysRun = false
		}

		if !job.RunBeforeMerge {
			job.RunBeforeMerge = true
		}
	}
}

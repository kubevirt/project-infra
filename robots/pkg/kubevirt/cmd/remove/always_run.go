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

package remove

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/release"

	"github.com/spf13/cobra"
	"k8s.io/test-infra/prow/config"
	"sigs.k8s.io/yaml"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/cmd/flags"
	kv_github "kubevirt.io/project-infra/robots/pkg/kubevirt/github"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/log"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/prowjobconfigs"
	"kubevirt.io/project-infra/robots/pkg/querier"
)

type removeAlwaysRunOptions struct {
	jobConfigPathKubevirtPresubmits string
}

func (o removeAlwaysRunOptions) Validate() error {
	if _, err := os.Stat(o.jobConfigPathKubevirtPresubmits); os.IsNotExist(err) {
		return fmt.Errorf("jobConfigPathKubevirtPresubmits is required: %v", err)
	}
	return nil
}

const (
	shortAlwaysRunUsage = "kubevirt remove always_run sets always_run to false on presubmit job definitions for kubevirt for unsupported kubevirtci providers"
)

var removeAlwaysRunCommand = &cobra.Command{
	Use:   "always_run",
	Short: shortAlwaysRunUsage,
	Long: fmt.Sprintf(`%s

For each of the sigs (%s)
it sets always_run to false on presubmit job definitions that contain
"old" k8s versions. From kubevirt standpoint an old k8s version is one
that is older than one of the three minor versions including the
currently released k8s version at the time of the check.

Example:

* k8s 1.22 is the current stable version
* job definitions exist for k8s 1.22, 1.21, 1.20
* presubmits for 1.22 are to run always and are required (aka optional: false)
* job definitions exist for 1.19 k8s version 

This will lead to always_run being set to false for each of the sigs presubmit jobs for 1.19

See kubevirt k8s version compatibility: https://github.com/kubevirt/kubevirt/blob/main/docs/kubernetes-compatibility.md#kubernetes-version-compatibility 
`, shortAlwaysRunUsage, strings.Join(prowjobconfigs.SigNames, ", ")),
	RunE: runAlwaysRunCommand,
}

var removeAlwaysRunOpts = removeAlwaysRunOptions{}

func RemoveAlwaysRunCommand() *cobra.Command {
	return removeAlwaysRunCommand
}

func init() {
	removeAlwaysRunCommand.PersistentFlags().StringVar(&removeAlwaysRunOpts.jobConfigPathKubevirtPresubmits, "job-config-path-kubevirt-presubmits", "", "The directory of the kubevirt presubmit job definitions")
}

func runAlwaysRunCommand(cmd *cobra.Command, args []string) error {
	err := flags.ParseFlags(cmd, args, removeAlwaysRunOpts)
	if err != nil {
		return err
	}

	ctx := context.Background()
	client, err := kv_github.NewGitHubClient(ctx)
	if err != nil {
		return err
	}

	jobConfig, err := config.ReadJobConfig(removeAlwaysRunOpts.jobConfigPathKubevirtPresubmits)
	if err != nil {
		return fmt.Errorf("failed to read jobconfig %s: %v", removeAlwaysRunOpts.jobConfigPathKubevirtPresubmits, err)
	}

	releases, _, err := client.Repositories.ListReleases(ctx, "kubernetes", "kubernetes", nil)
	if err != nil {
		return fmt.Errorf("failed to list releases: %v", err)
	}
	releases = querier.ValidReleases(releases)
	allReleases := release.AsSemVers(releases)
	latestMinorReleases := release.GetLatestMinorReleases(allReleases)
	if len(latestMinorReleases) < fourReleasesRequiredAtMinimum {
		log.Log().Info("Not enough minor releases found, nothing to do.")
		return nil
	}

	result, message := ensureSigJobsDoAlwaysRun(jobConfig, latestMinorReleases[0])
	if result != ALL_JOBS_DO_ALWAYS_RUN {
		log.Log().Infof("Not all presubmits for k8s %s do run always, nothing to do.\n%s", latestMinorReleases[0], message)
		return nil
	}

	targetReleaseSemver := latestMinorReleases[3]
	log.Log().Infof("Targeting release %v", targetReleaseSemver)

	updated := updatePresubmitsAlwaysRunField(&jobConfig, targetReleaseSemver)
	if !updated && !flags.Options.DryRun {
		log.Log().Info(fmt.Sprintf("presubmit jobs for %v weren't modified, nothing to do.", targetReleaseSemver))
		return nil
	}

	marshalledConfig, err := yaml.Marshal(&jobConfig)
	if err != nil {
		return fmt.Errorf("failed to marshall jobconfig %s: %v", removeAlwaysRunOpts.jobConfigPathKubevirtPresubmits, err)
	}

	if flags.Options.DryRun {
		_, err = os.Stdout.Write(marshalledConfig)
		if err != nil {
			return fmt.Errorf("failed to write jobconfig %s: %v", removeAlwaysRunOpts.jobConfigPathKubevirtPresubmits, err)
		}
		return nil
	}

	err = ioutil.WriteFile(removeAlwaysRunOpts.jobConfigPathKubevirtPresubmits, marshalledConfig, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to write jobconfig %s: %v", removeAlwaysRunOpts.jobConfigPathKubevirtPresubmits, err)
	}
	return nil
}

func updatePresubmitsAlwaysRunField(jobConfig *config.JobConfig, latestReleaseSemver *querier.SemVer) (updated bool) {
	jobsToCheck := map[string]string{}
	for _, sigName := range prowjobconfigs.SigNames {
		jobsToCheck[prowjobconfigs.CreatePresubmitJobName(latestReleaseSemver, sigName)] = ""
	}

	for index := range jobConfig.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig] {
		job := &jobConfig.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig][index]
		name := job.Name
		if _, exists := jobsToCheck[name]; !exists {
			continue
		}

		if job.AlwaysRun {
			job.AlwaysRun = false
			updated = true
		}
	}

	return
}

type latestJobsAlwaysRunCheckResult int

const (
	NOT_ALL_ALWAYS_RUN_JOBS_EXIST latestJobsAlwaysRunCheckResult = iota
	NOT_ALL_JOBS_DO_ALWAYS_RUN
	ALL_JOBS_DO_ALWAYS_RUN
)

func ensureSigJobsDoAlwaysRun(jobConfigKubevirtPresubmits config.JobConfig, release *querier.SemVer) (result latestJobsAlwaysRunCheckResult, message string) {
	result = ALL_JOBS_DO_ALWAYS_RUN
	messages := []string{}
	requiredJobNames := map[string]struct{}{}
	for _, sigName := range prowjobconfigs.SigNames {
		requiredJobNames[prowjobconfigs.CreatePresubmitJobName(release, sigName)] = struct{}{}
	}
	for _, presubmit := range jobConfigKubevirtPresubmits.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig] {
		if _, exists := requiredJobNames[presubmit.Name]; !exists {
			continue
		}
		delete(requiredJobNames, presubmit.Name)
		if !presubmit.AlwaysRun {
			messages = append(messages, fmt.Sprintf("job %s is not running always", presubmit.Name))
			result = NOT_ALL_JOBS_DO_ALWAYS_RUN
		}
	}
	if len(requiredJobNames) > 0 {
		messages = append(messages, fmt.Sprintf("jobs missing: %v", requiredJobNames))
		result = NOT_ALL_ALWAYS_RUN_JOBS_EXIST
	}
	message = strings.Join(messages, ", ")
	return
}

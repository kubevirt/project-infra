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
	"os"
	"strings"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/release"

	"github.com/google/go-github/github"
	"github.com/spf13/cobra"
	"k8s.io/test-infra/prow/config"
	"sigs.k8s.io/yaml"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/cmd/flags"
	github2 "kubevirt.io/project-infra/robots/pkg/kubevirt/github"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/log"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/prowjobconfigs"
	"kubevirt.io/project-infra/robots/pkg/querier"
)

type removeJobsOptions struct {
	jobConfigPathKubevirtPresubmits string
	jobConfigPathKubevirtPeriodics  string
}

func (o removeJobsOptions) Validate() error {
	if _, err := os.Stat(o.jobConfigPathKubevirtPresubmits); os.IsNotExist(err) {
		return fmt.Errorf("jobConfigPathKubevirtPresubmits is required: %v", err)
	}
	if _, err := os.Stat(o.jobConfigPathKubevirtPeriodics); os.IsNotExist(err) {
		return fmt.Errorf("jobConfigPathKubevirtPeriodics is required: %v", err)
	}
	return nil
}

const (
	shortUsage                    = "kubevirt remove jobs removes presubmit and periodic job definitions for kubevirt for unsupported kubevirtci providers"
	fourReleasesRequiredAtMinimum = 4
)

var removeJobsCommand = &cobra.Command{
	Use:   "jobs",
	Short: shortUsage,
	Long: fmt.Sprintf(`%s

For each of the sigs (%s)
it removes job definitions that contain "old"
k8s versions. From kubevirt standpoint an old k8s version is one
that is older than one of the three minor versions including the
currently released k8s version at the time of the check.

Example:

* k8s 1.22 is the current stable version
* job definitions exist for k8s 1.22, 1.21, 1.20
* presubmits for 1.22 are to run always and are required (aka optional: false)
* job definitions exist for 1.19 k8s version 

This will lead to each of the sigs periodic and presubmit jobs for 1.19 being removed

See kubevirt k8s version compatibility: https://github.com/kubevirt/kubevirt/blob/main/docs/kubernetes-compatibility.md#kubernetes-version-compatibility 
`, shortUsage, strings.Join(prowjobconfigs.SigNames, ", ")),
	RunE: run,
}

var removeJobsOpts = removeJobsOptions{}

func RemoveJobsCommand() *cobra.Command {
	return removeJobsCommand
}

func init() {
	removeJobsCommand.PersistentFlags().StringVar(&removeJobsOpts.jobConfigPathKubevirtPresubmits, "job-config-path-kubevirt-presubmits", "", "The directory of the kubevirt presubmit job definitions")
	removeJobsCommand.PersistentFlags().StringVar(&removeJobsOpts.jobConfigPathKubevirtPeriodics, "job-config-path-kubevirt-periodics", "", "The path to the kubevirt periodic job definitions")
}

func run(cmd *cobra.Command, args []string) error {
	err := flags.ParseFlags(cmd, args, removeJobsOpts)
	if err != nil {
		return err
	}

	ctx := context.Background()
	client, err := github2.NewGitHubClient(ctx)
	if err != nil {
		return err
	}

	releases, _, err := client.Repositories.ListReleases(ctx, "kubernetes", "kubernetes", nil)
	if err != nil {
		return fmt.Errorf("failed to list releases: %v", err)
	}
	releases = querier.ValidReleases(releases)
	return removeOldJobsIfNewOnesExist(releases)
}

func removeOldJobsIfNewOnesExist(releases []*github.RepositoryRelease) error {
	jobConfigKubevirtPresubmits, err := config.ReadJobConfig(removeJobsOpts.jobConfigPathKubevirtPresubmits)
	if err != nil {
		return fmt.Errorf("failed to read jobconfig %s: %v", removeJobsOpts.jobConfigPathKubevirtPresubmits, err)
	}

	latestMinorReleases := release.GetLatestMinorReleases(release.AsSemVers(releases))
	if len(latestMinorReleases) < fourReleasesRequiredAtMinimum {
		log.Log().Info("Not enough minor releases found, nothing to do.")
		return nil
	}

	result, message := ensureSigJobsAreRequired(jobConfigKubevirtPresubmits, latestMinorReleases[0])
	if result != ALL_JOBS_ARE_REQUIRED {
		log.Log().Infof("Not all presubmits for k8s %s are required, nothing to do.\n%s", latestMinorReleases[0], message)
		return nil
	}

	threeLatestRequiredMinorReleases := latestMinorReleases[0:3]
	jobsExist, message := ensureSigPresubmitJobsExistForReleases(jobConfigKubevirtPresubmits, threeLatestRequiredMinorReleases)
	if !jobsExist {
		log.Log().Infof("Not all required jobs for k8s versions %s exist, nothing to do.\n%s", threeLatestRequiredMinorReleases, message)
		return nil
	}

	jobConfigKubevirtPeriodics, err := config.ReadJobConfig(removeJobsOpts.jobConfigPathKubevirtPeriodics)
	if err != nil {
		return fmt.Errorf("failed to read jobconfig %s: %v", removeJobsOpts.jobConfigPathKubevirtPeriodics, err)
	}
	targetRelease := latestMinorReleases[3:4][0]
	if updated := deleteSigPeriodicJobsForRelease(&jobConfigKubevirtPeriodics, targetRelease); updated {
		err := writeJobConfig(&jobConfigKubevirtPeriodics, removeJobsOpts.jobConfigPathKubevirtPeriodics)
		if err != nil {
			return fmt.Errorf("failed to update periodics %s: %v", removeJobsOpts.jobConfigPathKubevirtPeriodics, err)
		}
	}

	if updated := deleteSigPresubmitJobsForRelease(&jobConfigKubevirtPresubmits, targetRelease); updated {
		err := writeJobConfig(&jobConfigKubevirtPresubmits, removeJobsOpts.jobConfigPathKubevirtPresubmits)
		if err != nil {
			return fmt.Errorf("failed to update presubmits %s: %v", removeJobsOpts.jobConfigPathKubevirtPresubmits, err)
		}
	}
	return nil
}

func deleteSigPresubmitJobsForRelease(jobConfig *config.JobConfig, targetRelease *querier.SemVer) (updated bool) {
	toDeleteJobNames := map[string]struct{}{}
	for _, sigName := range prowjobconfigs.SigNames {
		toDeleteJobNames[prowjobconfigs.CreatePresubmitJobName(targetRelease, sigName)] = struct{}{}
	}

	var newPresubmits []config.Presubmit

	for _, presubmit := range jobConfig.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig] {
		if _, exists := toDeleteJobNames[presubmit.Name]; exists {
			updated = true
			continue
		}
		newPresubmits = append(newPresubmits, presubmit)
	}

	if updated {
		jobConfig.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig] = newPresubmits
	}

	return
}

func deleteSigPeriodicJobsForRelease(jobConfig *config.JobConfig, release *querier.SemVer) (updated bool) {
	toDeleteJobNames := map[string]struct{}{}
	for _, sigName := range prowjobconfigs.SigNames {
		toDeleteJobNames[prowjobconfigs.CreatePeriodicJobName(release, sigName)] = struct{}{}
	}

	var newPeriodics []config.Periodic

	for _, periodic := range jobConfig.Periodics {
		if _, exists := toDeleteJobNames[periodic.Name]; exists {
			updated = true
			continue
		}
		newPeriodics = append(newPeriodics, periodic)
	}

	if updated {
		jobConfig.Periodics = newPeriodics
	}

	return
}

func writeJobConfig(jobConfigToWrite *config.JobConfig, jobConfigPath string) error {
	marshalledConfig, err := yaml.Marshal(jobConfigToWrite)
	if err != nil {
		return fmt.Errorf("failed to marshall jobconfig %s: %v", jobConfigPath, err)
	}

	if flags.Options.DryRun {
		_, err = os.Stdout.Write(marshalledConfig)
		if err != nil {
			return fmt.Errorf("failed to write jobconfig %s: %v", jobConfigPath, err)
		}
	} else {
		err = os.WriteFile(jobConfigPath, marshalledConfig, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to write jobconfig %s: %v", jobConfigPath, err)
		}
		log.Log().WithField("jobConfigPath", jobConfigPath).Info("Updated jobconfig file")
	}
	return nil
}

func ensureSigPresubmitJobsExistForReleases(jobConfigKubevirtPresubmits config.JobConfig, requiredReleases []*querier.SemVer) (allJobsExist bool, message string) {
	allJobsExist = true
	messages := []string{}

	requiredJobNames := map[string]struct{}{}
	for _, release := range requiredReleases {
		for _, sigName := range prowjobconfigs.SigNames {
			requiredJobNames[prowjobconfigs.CreatePresubmitJobName(release, sigName)] = struct{}{}
		}
	}

	for _, presubmit := range jobConfigKubevirtPresubmits.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig] {
		if _, exists := requiredJobNames[presubmit.Name]; !exists {
			continue
		}
		delete(requiredJobNames, presubmit.Name)
	}

	if len(requiredJobNames) > 0 {
		messages = append(messages, fmt.Sprintf("jobs missing: %v", requiredJobNames))
		allJobsExist = false
	}

	message = strings.Join(messages, ", ")
	return allJobsExist, message
}

type latestJobsRequiredCheckResult int

const (
	NOT_ALL_JOBS_EXIST latestJobsRequiredCheckResult = iota
	NOT_ALL_JOBS_ARE_REQUIRED
	ALL_JOBS_ARE_REQUIRED
)

func ensureSigJobsAreRequired(jobConfigKubevirtPresubmits config.JobConfig, release *querier.SemVer) (result latestJobsRequiredCheckResult, message string) {
	result = ALL_JOBS_ARE_REQUIRED
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
			result = NOT_ALL_JOBS_ARE_REQUIRED
		}
		if presubmit.Optional {
			messages = append(messages, fmt.Sprintf("job %s is optional", presubmit.Name))
			result = NOT_ALL_JOBS_ARE_REQUIRED
		}
	}
	if len(requiredJobNames) > 0 {
		messages = append(messages, fmt.Sprintf("jobs missing: %v", requiredJobNames))
		result = NOT_ALL_JOBS_EXIST
	}
	message = strings.Join(messages, ", ")
	return
}

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

package copy

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/google/go-github/github"
	"k8s.io/test-infra/prow/config"
	"sigs.k8s.io/yaml"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/cmd/flags"
	github2 "kubevirt.io/project-infra/robots/pkg/kubevirt/github"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/log"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/prowjobconfigs"
	"kubevirt.io/project-infra/robots/pkg/querier"
)

const (
	shortUse                      = "kubevirt copy jobs creates copies of the periodic and presubmit SIG jobs for latest kubevirtci providers"
	sourceAndTargetReleaseDoExist = 2
)

type copyJobOptions struct {
	jobConfigPathKubevirtPresubmits string
	jobConfigPathKubevirtPeriodics  string
}

func (o copyJobOptions) Validate() error {
	if _, err := os.Stat(o.jobConfigPathKubevirtPresubmits); os.IsNotExist(err) {
		return fmt.Errorf("jobConfigPathKubevirtPresubmits is required: %v", err)
	}
	if _, err := os.Stat(o.jobConfigPathKubevirtPeriodics); os.IsNotExist(err) {
		return fmt.Errorf("jobConfigPathKubevirtPeriodics is required: %v", err)
	}
	return nil
}

var copyJobsOpts = copyJobOptions{}

var copyJobsCommand = &cobra.Command{
	Use:   "jobs",
	Short: shortUse,
	Long: fmt.Sprintf(`%s

For each of the sigs (%s)
it checks whether a job to run with the latest k8s version exists.
If not, it copies the existing job for the previously latest kubevirtci provider and
adjusts it to run with an eventually soon-to-be-existing new kubevirtci provider.

Presubmit jobs will be created with

	always_run: false
	optional: true

to avoid them failing all the time until the new provider is integrated into kubevirt/kubevirt.
`, shortUse, strings.Join(prowjobconfigs.SigNames, ", ")),
	RunE: run,
}

func CopyJobsCommand() *cobra.Command {
	return copyJobsCommand
}

func init() {
	copyJobsCommand.PersistentFlags().StringVar(&copyJobsOpts.jobConfigPathKubevirtPresubmits, "job-config-path-kubevirt-presubmits", "", "The path to the kubevirt presubmit job definitions")
	copyJobsCommand.PersistentFlags().StringVar(&copyJobsOpts.jobConfigPathKubevirtPeriodics, "job-config-path-kubevirt-periodics", "", "The path to the kubevirt periodic job definitions")
}

func run(cmd *cobra.Command, args []string) error {
	err := flags.ParseFlags(cmd, args, copyJobsOpts)
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
		return err
	}
	releases = querier.ValidReleases(releases)
	targetRelease, sourceRelease, err := getSourceAndTargetRelease(releases)
	if err != nil {
		log.Log().WithError(err).Info("Cannot determine source and target Release.")
		return nil
	}

	jobConfigs := map[string]func(*config.JobConfig, *querier.SemVer, *querier.SemVer) bool{
		copyJobsOpts.jobConfigPathKubevirtPresubmits: func(jobConfig *config.JobConfig, latestReleaseSemver *querier.SemVer, secondLatestReleaseSemver *querier.SemVer) bool {
			return copyPresubmitJobsForNewProvider(jobConfig, latestReleaseSemver, secondLatestReleaseSemver)
		},
		copyJobsOpts.jobConfigPathKubevirtPeriodics: func(jobConfig *config.JobConfig, latestReleaseSemver *querier.SemVer, secondLatestReleaseSemver *querier.SemVer) bool {
			return copyPeriodicJobsForNewProvider(jobConfig, latestReleaseSemver, secondLatestReleaseSemver)
		},
	}
	for jobConfigPath, jobConfigCopyFunc := range jobConfigs {
		jobConfig, err := config.ReadJobConfig(jobConfigPath)
		if err != nil {
			return fmt.Errorf("failed to read jobconfig %s: %v", jobConfigPath, err)
		}

		updated := jobConfigCopyFunc(&jobConfig, targetRelease, sourceRelease)
		if !updated && !flags.Options.DryRun {
			log.Log().WithField("jobConfigPath", jobConfigPath).Info(fmt.Sprintf("presubmit jobs for %v weren't modified, nothing to do.", targetRelease))
			continue
		}

		marshalledConfig, err := yaml.Marshal(&jobConfig)
		if err != nil {
			return fmt.Errorf("failed to marshall jobconfig %s: %v", jobConfigPath, err)
		}

		if flags.Options.DryRun {
			_, err = os.Stdout.Write(marshalledConfig)
			if err != nil {
				return fmt.Errorf("failed to write jobconfig %s to stdout: %v", jobConfigPath, err)
			}
			continue
		}

		err = os.WriteFile(jobConfigPath, marshalledConfig, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to write jobconfig %s: %v", jobConfigPath, err)
		}
	}
	return nil
}

func getSourceAndTargetRelease(releases []*github.RepositoryRelease) (targetRelease *querier.SemVer, sourceRelease *querier.SemVer, err error) {
	if len(releases) < sourceAndTargetReleaseDoExist {
		err = fmt.Errorf("less than two releases")
		return
	}
	targetRelease = querier.ParseRelease(releases[0])
	for _, release := range releases[1:] {
		nextRelease := querier.ParseRelease(release)
		if nextRelease.Minor < targetRelease.Minor {
			sourceRelease = nextRelease
			break
		}
	}
	if sourceRelease == nil {
		err = fmt.Errorf("no source Release found")
	}
	return
}

func copyPresubmitJobsForNewProvider(jobConfig *config.JobConfig, targetProviderReleaseSemver *querier.SemVer, sourceProviderReleaseSemver *querier.SemVer) (updated bool) {
	allPresubmitJobs := map[string]config.Presubmit{}
	for index := range jobConfig.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig] {
		job := jobConfig.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig][index]
		allPresubmitJobs[job.Name] = job
	}

	for _, sigName := range prowjobconfigs.SigNames {
		targetJobName := prowjobconfigs.CreatePresubmitJobName(targetProviderReleaseSemver, sigName)
		sourceJobName := prowjobconfigs.CreatePresubmitJobName(sourceProviderReleaseSemver, sigName)

		if _, exists := allPresubmitJobs[targetJobName]; exists {
			log.Log().WithField("targetJobName", targetJobName).WithField("sourceJobName", sourceJobName).Info("Target job exists, nothing to do")
			continue
		}

		if _, exists := allPresubmitJobs[sourceJobName]; !exists {
			log.Log().WithField("targetJobName", targetJobName).WithField("sourceJobName", sourceJobName).Warn("Source job does not exist, can't copy job definition!")
			continue
		}

		log.Log().WithField("targetJobName", targetJobName).WithField("sourceJobName", sourceJobName).Info("Copying source to target job")

		newJob := config.Presubmit{}
		newJob.Annotations = make(map[string]string)
		for k, v := range allPresubmitJobs[sourceJobName].Annotations {
			newJob.Annotations[k] = v
		}
		newJob.Cluster = allPresubmitJobs[sourceJobName].Cluster
		newJob.Decorate = allPresubmitJobs[sourceJobName].Decorate
		newJob.DecorationConfig = allPresubmitJobs[sourceJobName].DecorationConfig.DeepCopy()
		copy(newJob.ExtraRefs, allPresubmitJobs[sourceJobName].ExtraRefs)
		newJob.Labels = make(map[string]string)
		for k, v := range allPresubmitJobs[sourceJobName].Labels {
			newJob.Labels[k] = v
		}
		newJob.MaxConcurrency = allPresubmitJobs[sourceJobName].MaxConcurrency
		newJob.Spec = allPresubmitJobs[sourceJobName].Spec.DeepCopy()
		newJob.Brancher.SkipBranches = allPresubmitJobs[sourceJobName].Brancher.SkipBranches
		newJob.Brancher.Branches = allPresubmitJobs[sourceJobName].Brancher.Branches

		newJob.AlwaysRun = false
		for index, envVar := range newJob.Spec.Containers[0].Env {
			if envVar.Name != "TARGET" {
				continue
			}
			newEnvVar := *envVar.DeepCopy()
			newEnvVar.Value = prowjobconfigs.CreateTargetValue(targetProviderReleaseSemver, sigName)
			newJob.Spec.Containers[0].Env[index] = newEnvVar
			break
		}
		newJob.Name = targetJobName
		newJob.Optional = true
		jobConfig.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig] = append(jobConfig.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig], newJob)

		updated = true
	}

	return
}

func copyPeriodicJobsForNewProvider(jobConfig *config.JobConfig, targetProviderReleaseSemver *querier.SemVer, sourceProviderReleaseSemver *querier.SemVer) (updated bool) {
	allPeriodicJobs := map[string]config.Periodic{}
	for index := range jobConfig.Periodics {
		job := jobConfig.Periodics[index]
		allPeriodicJobs[job.Name] = job
	}

	for _, sigName := range prowjobconfigs.SigNames {
		targetJobName := prowjobconfigs.CreatePeriodicJobName(targetProviderReleaseSemver, sigName)
		sourceJobName := prowjobconfigs.CreatePeriodicJobName(sourceProviderReleaseSemver, sigName)

		if _, exists := allPeriodicJobs[targetJobName]; exists {
			log.Log().WithField("targetJobName", targetJobName).WithField("sourceJobName", sourceJobName).Info("Target job exists, nothing to do")
			continue
		}

		if _, exists := allPeriodicJobs[sourceJobName]; !exists {
			log.Log().WithField("targetJobName", targetJobName).WithField("sourceJobName", sourceJobName).Warn("Source job does not exist, can't copy job definition!")
			continue
		}

		log.Log().WithField("targetJobName", targetJobName).WithField("sourceJobName", sourceJobName).Info("Copying source to target job")

		newJob := config.Periodic{}
		newJob.Annotations = make(map[string]string)
		for k, v := range allPeriodicJobs[sourceJobName].Annotations {
			newJob.Annotations[k] = v
		}
		newJob.Cluster = allPeriodicJobs[sourceJobName].Cluster
		newJob.Cron = prowjobconfigs.AdvanceCronExpression(allPeriodicJobs[sourceJobName].Cron)
		newJob.Decorate = allPeriodicJobs[sourceJobName].Decorate
		newJob.DecorationConfig = allPeriodicJobs[sourceJobName].DecorationConfig.DeepCopy()
		copy(newJob.ExtraRefs, allPeriodicJobs[sourceJobName].ExtraRefs)
		newJob.Labels = make(map[string]string)
		for k, v := range allPeriodicJobs[sourceJobName].Labels {
			newJob.Labels[k] = v
		}
		newJob.MaxConcurrency = allPeriodicJobs[sourceJobName].MaxConcurrency
		newJob.ReporterConfig = allPeriodicJobs[sourceJobName].ReporterConfig.DeepCopy()
		newJob.Spec = allPeriodicJobs[sourceJobName].Spec.DeepCopy()

		for _, extraRef := range allPeriodicJobs[sourceJobName].UtilityConfig.ExtraRefs {
			newJob.UtilityConfig.ExtraRefs = append(newJob.UtilityConfig.ExtraRefs, extraRef)
		}

		for containerIndex, container := range newJob.Spec.Containers {
			for envVarIndex, envVar := range container.Env {
				if envVar.Name != "TARGET" {
					continue
				}
				newEnvVar := *envVar.DeepCopy()
				newEnvVar.Value = prowjobconfigs.CreateTargetValue(targetProviderReleaseSemver, sigName)
				newJob.Spec.Containers[containerIndex].Env[envVarIndex] = newEnvVar
				break
			}
		}
		newJob.Name = targetJobName
		jobConfig.Periodics = append(jobConfig.Periodics, newJob)

		updated = true
	}

	return
}

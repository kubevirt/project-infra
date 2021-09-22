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
	"github.com/spf13/cobra"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"k8s.io/test-infra/prow/config"
	"sigs.k8s.io/yaml"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/cmd/flags"
	github2 "kubevirt.io/project-infra/robots/pkg/kubevirt/github"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/jobconfig"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/log"
	"kubevirt.io/project-infra/robots/pkg/querier"
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

const shortUse = "kubevirt copy jobs creates copies of the periodic and presubmit SIG jobs for latest kubevirtci providers"

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
`, shortUse, strings.Join(jobconfig.SigNames, ", ")),
	Run: run,
}

func CopyJobsCommand() *cobra.Command {
	return copyJobsCommand
}

func init() {
	copyJobsCommand.PersistentFlags().StringVar(&copyJobsOpts.jobConfigPathKubevirtPresubmits, "job-config-path-kubevirt-presubmits", "", "The path to the kubevirt presubmit job definitions")
	copyJobsCommand.PersistentFlags().StringVar(&copyJobsOpts.jobConfigPathKubevirtPeriodics, "job-config-path-kubevirt-periodics", "", "The path to the kubevirt periodic job definitions")
}

func run(cmd *cobra.Command, args []string) {
	flags.ParseFlagsOrExit(cmd, args, copyJobsOpts)

	ctx := context.Background()
	client := github2.NewGitHubClient(ctx)

	releases, _, err := client.Repositories.ListReleases(ctx, "kubernetes", "kubernetes", nil)
	if err != nil {
		log.Log().Panicln(err)
	}
	releases = querier.ValidReleases(releases)
	if len(releases) < 2 {
		log.Log().Info("No two releases found, nothing to do.")
		os.Exit(0)
	}

	targetRelease, sourceRelease, err := getSourceAndTargetRelease(releases)
	if err != nil {
		log.Log().WithError(err).Info("Cannot determine source and target release.")
		os.Exit(0)
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
			log.Log().WithField("jobConfigPath", jobConfigPath).WithError(err).Fatal("Failed to read jobconfig")
		}

		updated := jobConfigCopyFunc(&jobConfig, targetRelease, sourceRelease)
		if !updated && !flags.Options.DryRun {
			log.Log().WithField("jobConfigPath", jobConfigPath).Info(fmt.Sprintf("presubmit jobs for %v weren't modified, nothing to do.", targetRelease))
			continue
		}

		marshalledConfig, err := yaml.Marshal(&jobConfig)
		if err != nil {
			log.Log().WithField("jobConfigPath", jobConfigPath).WithError(err).Error("Failed to marshall jobconfig")
		}

		if flags.Options.DryRun {
			_, err = os.Stdout.Write(marshalledConfig)
			if err != nil {
				log.Log().WithField("jobConfigPath", jobConfigPath).WithError(err).Error("Failed to write jobconfig")
			}
			continue
		}

		err = os.WriteFile(jobConfigPath, marshalledConfig, os.ModePerm)
		if err != nil {
			log.Log().WithField("jobConfigPath", jobConfigPath).WithError(err).Error("Failed to write jobconfig")
		}
	}
}

func getSourceAndTargetRelease(releases []*github.RepositoryRelease) (targetRelease *querier.SemVer, sourceRelease *querier.SemVer, err error) {
	if len(releases) < 2 {
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
		err = fmt.Errorf("no source release found")
	}
	return
}

func copyPresubmitJobsForNewProvider(jobConfig *config.JobConfig, targetProviderReleaseSemver *querier.SemVer, sourceProviderReleaseSemver *querier.SemVer) (updated bool) {
	allPresubmitJobs := map[string]config.Presubmit{}
	for index := range jobConfig.PresubmitsStatic[jobconfig.OrgAndRepoForJobConfig] {
		job := jobConfig.PresubmitsStatic[jobconfig.OrgAndRepoForJobConfig][index]
		allPresubmitJobs[job.Name] = job
	}

	for _, sigName := range jobconfig.SigNames {
		targetJobName := jobconfig.CreatePresubmitJobName(targetProviderReleaseSemver, sigName)
		sourceJobName := jobconfig.CreatePresubmitJobName(sourceProviderReleaseSemver, sigName)

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

		newJob.AlwaysRun = false
		for index, envVar := range newJob.Spec.Containers[0].Env {
			if envVar.Name != "TARGET" {
				continue
			}
			newEnvVar := *envVar.DeepCopy()
			newEnvVar.Value = jobconfig.CreateTargetValue(targetProviderReleaseSemver, sigName)
			newJob.Spec.Containers[0].Env[index] = newEnvVar
			break
		}
		newJob.Name = targetJobName
		newJob.Optional = true
		jobConfig.PresubmitsStatic[jobconfig.OrgAndRepoForJobConfig] = append(jobConfig.PresubmitsStatic[jobconfig.OrgAndRepoForJobConfig], newJob)

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

	for _, sigName := range jobconfig.SigNames {
		targetJobName := jobconfig.CreatePeriodicJobName(targetProviderReleaseSemver, sigName)
		sourceJobName := jobconfig.CreatePeriodicJobName(sourceProviderReleaseSemver, sigName)

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
		newJob.Cron = jobconfig.AdvanceCronExpression(allPeriodicJobs[sourceJobName].Cron)
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

		for index, envVar := range newJob.Spec.Containers[0].Env {
			if envVar.Name != "TARGET" {
				continue
			}
			newEnvVar := *envVar.DeepCopy()
			newEnvVar.Value = jobconfig.CreateTargetValue(targetProviderReleaseSemver, sigName)
			newJob.Spec.Containers[0].Env[index] = newEnvVar
			break
		}
		newJob.Name = targetJobName
		jobConfig.Periodics = append(jobConfig.Periodics, newJob)

		updated = true
	}

	return
}

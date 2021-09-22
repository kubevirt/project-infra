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
	"github.com/spf13/cobra"
	"k8s.io/test-infra/prow/config"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/cmd/flags"
	github2 "kubevirt.io/project-infra/robots/pkg/kubevirt/github"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/jobconfig"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/log"
	"kubevirt.io/project-infra/robots/pkg/querier"
	"os"
	"strings"
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

const shortUsage = "kubevirt remove jobs removes presubmit and periodic job definitions for kubevirt for unsupported kubevirtci providers"

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
`, shortUsage, strings.Join(jobconfig.SigNames, ", ")),
	Run: run,
}

var removeJobsOpts = removeJobsOptions{}

func RemoveJobsCommand() *cobra.Command {
	return removeJobsCommand
}

func init() {
	removeJobsCommand.PersistentFlags().StringVar(&removeJobsOpts.jobConfigPathKubevirtPresubmits, "job-config-path-kubevirt-presubmits", "", "The directory of the kubevirt presubmit job definitions")
	removeJobsCommand.PersistentFlags().StringVar(&removeJobsOpts.jobConfigPathKubevirtPeriodics, "job-config-path-kubevirt-periodics", "", "The path to the kubevirt periodic job definitions")
}

func run(cmd *cobra.Command, args []string) {
	flags.ParseFlagsOrExit(cmd, args, removeJobsOpts)

	ctx := context.Background()
	client := github2.NewGitHubClient(ctx)

	releases, _, err := client.Repositories.ListReleases(ctx, "kubernetes", "kubernetes", nil)
	if err != nil {
		log.Log().Panicln(err)
	}
	releases = querier.ValidReleases(releases)
	if len(releases) < 4 {
		log.Log().Info("Not enough releases found, nothing to do.")
		os.Exit(0)
	}

	jobConfigKubevirtPresubmits, err := config.ReadJobConfig(removeJobsOpts.jobConfigPathKubevirtPresubmits)
	if err != nil {
		log.Log().WithField("jobConfigPathKubevirtPresubmits", removeJobsOpts.jobConfigPathKubevirtPresubmits).WithError(err).Fatal("Failed to read jobconfig")
	}

	result, message := ensureLatestJobsAreRequired(jobConfigKubevirtPresubmits, querier.ParseRelease(releases[0]))
	if result != ALL_JOBS_ARE_REQUIRED {
		log.Log().Infof("Not all presubmits for k8s %s are required, nothing to do.\n%s", releases[0], message)
		os.Exit(0)
	}

	var requiredReleases []*querier.SemVer
	for _, release := range releases[0:3] {
		requiredReleases = append(requiredReleases, querier.ParseRelease(release))
	}
	jobsExist, message := ensureJobsExistForReleases(jobConfigKubevirtPresubmits, requiredReleases)
	if !jobsExist {
		log.Log().Infof("Not all required jobs for k8s versions %s exist, nothing to do.\n%s", requiredReleases, message)
		os.Exit(0)
	}
	panic(fmt.Errorf("TODO"))
}

func ensureJobsExistForReleases(jobConfigKubevirtPresubmits config.JobConfig, requiredReleases []*querier.SemVer) (allJobsExist bool, message string) {
	allJobsExist = true
	messages := []string{}

	requiredJobNames := map[string]struct{}{}
	for _, release := range requiredReleases {
		for _, sigName := range jobconfig.SigNames {
			requiredJobNames[jobconfig.CreatePresubmitJobName(release, sigName)] = struct{}{}
		}
	}

	for _, presubmit := range jobConfigKubevirtPresubmits.PresubmitsStatic[jobconfig.OrgAndRepoForJobConfig] {
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

func ensureLatestJobsAreRequired(jobConfigKubevirtPresubmits config.JobConfig, release *querier.SemVer) (result latestJobsRequiredCheckResult, message string) {
	result = ALL_JOBS_ARE_REQUIRED
	messages := []string{}
	requiredJobNames := map[string]struct{}{}
	for _, sigName := range jobconfig.SigNames {
		requiredJobNames[jobconfig.CreatePresubmitJobName(release, sigName)] = struct{}{}
	}
	for _, presubmit := range jobConfigKubevirtPresubmits.PresubmitsStatic[jobconfig.OrgAndRepoForJobConfig] {
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

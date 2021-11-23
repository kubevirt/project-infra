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

package check

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/spf13/cobra"
	"k8s.io/test-infra/prow/config"
	"sigs.k8s.io/yaml"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/cmd/flags"
	github2 "kubevirt.io/project-infra/robots/pkg/kubevirt/github"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/log"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/prowjobconfigs"
	"kubevirt.io/project-infra/robots/pkg/querier"
)

const (
	shortUse = "kubevirt check providers checks the usage of kubevirtci providers for periodic and presubmit jobs for kubevirt/kubevirt"
)

type checkProvidersOptions struct {
	jobConfigPathKubevirtPresubmits string
	jobConfigPathKubevirtPeriodics  string
	outputFile                      string
	overwrite                       bool
	failOnUnsupported               bool
}

func (o checkProvidersOptions) Validate() error {
	if _, err := os.Stat(o.jobConfigPathKubevirtPresubmits); os.IsNotExist(err) {
		return fmt.Errorf("jobConfigPathKubevirtPresubmits is required: %v", err)
	}
	if _, err := os.Stat(o.jobConfigPathKubevirtPeriodics); os.IsNotExist(err) {
		return fmt.Errorf("jobConfigPathKubevirtPeriodics is required: %v", err)
	}
	return nil
}

var checkProvidersOpts = checkProvidersOptions{}

var checkProvidersCommand = &cobra.Command{
	Use:   "providers",
	Short: shortUse,
	Long: fmt.Sprintf(`%s

For each of the periodics and presubmits for kubevirt/kubevirt it matches the TARGET env variable value of all containers specs against the provider name pattern and records all matches.
It then generates a list of used providers, separated in unsupported and supported ones.
`, shortUse),
	RunE: run,
}

const providerNamePattern = "k8s-%s.%s"

var providerRegex *regexp.Regexp

func CheckProvidersCommand() *cobra.Command {
	return checkProvidersCommand
}

func init() {
	checkProvidersCommand.PersistentFlags().StringVar(&checkProvidersOpts.jobConfigPathKubevirtPresubmits, "job-config-path-kubevirt-presubmits", "", "The path to the kubevirt presubmit job definitions")
	checkProvidersCommand.PersistentFlags().StringVar(&checkProvidersOpts.jobConfigPathKubevirtPeriodics, "job-config-path-kubevirt-periodics", "", "The path to the kubevirt periodic job definitions")
	checkProvidersCommand.PersistentFlags().StringVar(&checkProvidersOpts.outputFile, "output-file", "", "Path to output file, if not given, a temporary file will be used")
	checkProvidersCommand.PersistentFlags().BoolVar(&checkProvidersOpts.overwrite, "overwrite", false, "Whether to overwrite output file")
	checkProvidersCommand.PersistentFlags().BoolVar(&checkProvidersOpts.failOnUnsupported, "fail-on-unsupported-provider-usage", true, "Whether to exit with non zero exit code in case an unsupported provider usage is detected")

	providerRegex = regexp.MustCompile("(k8s-[0-9.]+)")
}

type checkProvidersUsed struct {
	Supported   map[string][]string `json:"supported,omitempty"`
	Unsupported map[string][]string `json:"unsupported,omitempty"`
}

func run(cmd *cobra.Command, args []string) error {
	err := flags.ParseFlags(cmd, args, checkProvidersOpts)
	if err != nil {
		return err
	}

	if checkProvidersOpts.outputFile == "" {
		outputFile, err := os.CreateTemp("", "kubevirt-check-providers-*.yaml")
		if err != nil {
			return fmt.Errorf("failed to write output: %v", err)
		}
		checkProvidersOpts.outputFile = outputFile.Name()
	} else {
		stat, err := os.Stat(checkProvidersOpts.outputFile)
		if err != nil && err != os.ErrNotExist {
			return fmt.Errorf("failed to write output: %v", err)
		}
		if stat.IsDir() {
			return fmt.Errorf("failed to write output, file %s is a directory", checkProvidersOpts.outputFile)
		}
		if err == nil && !checkProvidersOpts.overwrite {
			log.Log().Fatal(fmt.Errorf("failed to write output, file %s exists", checkProvidersOpts.outputFile))
		}
	}
	log.Log().Infof("Will write output to file %s", checkProvidersOpts.outputFile)

	ctx := context.Background()
	client, err := github2.NewGitHubClient(ctx)
	if err != nil {
		return err
	}

	jobConfigsToExtractFuncs := map[string]func(*config.JobConfig, map[string][]string){
		checkProvidersOpts.jobConfigPathKubevirtPresubmits: func(jobConfig *config.JobConfig, result map[string][]string) {
			extractUsedProvidersFromPresubmitJobs(jobConfig, result)
		},
		checkProvidersOpts.jobConfigPathKubevirtPeriodics: func(jobConfig *config.JobConfig, result map[string][]string) {
			extractUsedProvidersFromPeriodicJobs(jobConfig, result)
		},
	}

	allProviderNamesToJobNames := map[string][]string{}
	for jobConfigPath, usedProvidersFromJobsExtractFunc := range jobConfigsToExtractFuncs {
		jobConfig, err := config.ReadJobConfig(jobConfigPath)
		if err != nil {
			return fmt.Errorf("failed to read jobconfig %s: %v", jobConfigPath, err)
		}
		usedProvidersFromJobsExtractFunc(&jobConfig, allProviderNamesToJobNames)
	}

	releases, _, err := client.Repositories.ListReleases(ctx, "kubernetes", "kubernetes", nil)
	if err != nil {
		return err
	}
	releases = querier.ValidReleases(releases)
	supportedProviderVersions := querier.LastThreeMinor(1, releases)

	providersUsed := checkProvidersUsed{Supported: map[string][]string{}, Unsupported: map[string][]string{}}
	for _, supportedProviderName := range supportedProviderVersions {
		semVer := querier.ParseRelease(supportedProviderName)
		providerName := fmt.Sprintf(providerNamePattern, semVer.Major, semVer.Minor)
		providersUsed.Supported[providerName] = allProviderNamesToJobNames[providerName]
		delete(allProviderNamesToJobNames, providerName)
	}
	for unsupportedProviderName, jobNames := range allProviderNamesToJobNames {
		providersUsed.Unsupported[unsupportedProviderName] = jobNames
	}

	marshalled, err := yaml.Marshal(providersUsed)
	if err != nil {
		return fmt.Errorf("failed to marshall output %s: %v", checkProvidersOpts.outputFile, err)
	}
	err = os.WriteFile(checkProvidersOpts.outputFile, marshalled, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to write output %s: %v", checkProvidersOpts.outputFile, err)
	}

	if checkProvidersOpts.failOnUnsupported && len(providersUsed.Unsupported) > 0 {
		return fmt.Errorf("usage of unsupported providers detected: %v", providersUsed.Unsupported)
	}

	return nil
}

func extractUsedProvidersFromPeriodicJobs(jobConfig *config.JobConfig, result map[string][]string) {
	for index := range jobConfig.Periodics {
		appendProviderNamesToJobNames((&jobConfig.Periodics[index]).JobBase, result)
	}
}

func extractUsedProvidersFromPresubmitJobs(jobConfig *config.JobConfig, result map[string][]string) {
	for index := range jobConfig.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig] {
		appendProviderNamesToJobNames((&jobConfig.PresubmitsStatic[prowjobconfigs.OrgAndRepoForJobConfig][index]).JobBase, result)
	}
}

func appendProviderNamesToJobNames(base config.JobBase, result map[string][]string) {
	name := base.Name
	var providerNames []string
	for _, container := range base.Spec.Containers {
		for _, env := range container.Env {
			if env.Name != "TARGET" {
				continue
			}
			if !providerRegex.MatchString(env.Value) {
				continue
			}
			var submatches []string
			submatches = providerRegex.FindStringSubmatch(name)
			providerNames = append(providerNames, submatches[1])
		}
	}
	for _, providerName := range providerNames {
		if _, exists := result[providerName]; exists {
			result[providerName] = append(result[providerName], name)
		} else {
			result[providerName] = []string{name}
		}
	}
}

/*
 * Copyright 2026 The KubeVirt Authors.
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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"sigs.k8s.io/prow/pkg/config"

	"kubevirt.io/project-infra/pkg/kubevirt/cmd/flags"
	"kubevirt.io/project-infra/pkg/kubevirt/prowjobconfigs"
)

type checkPresubmitsOptions struct {
	jobConfigPathKubevirtPresubmits string
}

func (o checkPresubmitsOptions) Validate() error {
	if _, err := os.Stat(o.jobConfigPathKubevirtPresubmits); os.IsNotExist(err) {
		return fmt.Errorf("jobConfigPathKubevirtPresubmits is required: %v", err)
	}
	return nil
}

var checkPresubmitsOpts = checkPresubmitsOptions{}

var checkPresubmitsCommand = &cobra.Command{
	Use:   "presubmits",
	Short: "kubevirt check presubmits validates sig-compute and sig-network presubmit invariants",
	Long: `kubevirt check presubmits validates sig-compute and sig-network presubmit invariants

For the latest Kubernetes version found in the presubmit job definitions, it checks that
the sig-compute TARGET configuration is aligned and that the sig-network lane
configuration stays coherent with the sig-network-smoke lane.`,
	RunE: runCheckPresubmits,
}

func CheckPresubmitsCommand() *cobra.Command {
	return checkPresubmitsCommand
}

func init() {
	checkPresubmitsCommand.PersistentFlags().StringVar(&checkPresubmitsOpts.jobConfigPathKubevirtPresubmits, "job-config-path-kubevirt-presubmits", "", "The path to the kubevirt presubmit job definitions")
}

func runCheckPresubmits(cmd *cobra.Command, args []string) error {
	err := flags.ParseFlags(cmd, args, checkPresubmitsOpts)
	if err != nil {
		return err
	}

	jobConfig, err := config.ReadJobConfig(checkPresubmitsOpts.jobConfigPathKubevirtPresubmits)
	if err != nil {
		return fmt.Errorf("failed to read jobconfig %s: %v", checkPresubmitsOpts.jobConfigPathKubevirtPresubmits, err)
	}

	var errs []string
	errs = append(errs, validateSigComputeTargets(&jobConfig)...)
	errs = append(errs, validateSigNetworkSkipConformanceEnv(&jobConfig)...)

	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintln(cmd.OutOrStderr(), e)
		}
		return fmt.Errorf("presubmit validation failed")
	}

	return nil
}

func validateSigComputeTargets(jobConfig *config.JobConfig) []string {
	jobs := prowjobconfigs.CollectSigComputeJobs(jobConfig)
	if len(jobs) == 0 {
		return []string{"no sig-compute jobs found"}
	}

	latestVersion, err := prowjobconfigs.FindLatestK8sVersionFromJobs(jobs)
	if err != nil {
		return []string{err.Error()}
	}

	serialJobName := fmt.Sprintf("pull-kubevirt-e2e-k8s-%s-sig-compute-serial", latestVersion)
	bareJobName := fmt.Sprintf("pull-kubevirt-e2e-k8s-%s-sig-compute", latestVersion)
	expectedSerialTarget := fmt.Sprintf("k8s-%s-sig-compute-serial", latestVersion)
	expectedParallelTarget := fmt.Sprintf("k8s-%s-sig-compute-parallel", latestVersion)

	var errs []string
	for _, job := range jobs {
		switch job.Name {
		case serialJobName:
			if job.Target != expectedSerialTarget {
				errs = append(errs, fmt.Sprintf("job %q has TARGET %q, expected %q", job.Name, job.Target, expectedSerialTarget))
			}
		case bareJobName:
			if job.Target != expectedParallelTarget {
				errs = append(errs, fmt.Sprintf("job %q has TARGET %q, expected %q", job.Name, job.Target, expectedParallelTarget))
			}
		}
	}

	return errs
}

func validateSigNetworkSkipConformanceEnv(jobConfig *config.JobConfig) []string {
	jobs := prowjobconfigs.CollectSigNetworkJobs(jobConfig)
	if len(jobs) == 0 {
		return []string{"no sig-network jobs found"}
	}

	var errs []string
	var smokeVersion string
	var skipConformanceVersions []string

	for _, job := range jobs {
		if job.IsSmoke {
			if smokeVersion != "" {
				errs = append(errs, fmt.Sprintf("multiple sig-network-smoke jobs found: %q and k8s-%s", job.Name, smokeVersion))
				continue
			}
			smokeVersion = job.K8sVersion
			continue
		}

		if job.SkipConformanceEnv == "true" {
			skipConformanceVersions = append(skipConformanceVersions, job.K8sVersion)
			continue
		}
		if job.SkipConformanceEnv != "" {
			errs = append(errs, fmt.Sprintf("job %q has SKIP_CONFORMANCE=%q, expected %q", job.Name, job.SkipConformanceEnv, "true"))
		}
	}

	if smokeVersion == "" {
		errs = append(errs, "no sig-network-smoke job found")
	}

	switch len(skipConformanceVersions) {
	case 0:
		errs = append(errs, "no plain sig-network job has SKIP_CONFORMANCE=true")
	case 1:
		if smokeVersion != "" && skipConformanceVersions[0] != smokeVersion {
			errs = append(errs, fmt.Sprintf("sig-network-smoke runs on k8s-%s, but SKIP_CONFORMANCE=true is set on sig-network k8s-%s", smokeVersion, skipConformanceVersions[0]))
		}
	default:
		errs = append(errs, fmt.Sprintf("multiple plain sig-network jobs have SKIP_CONFORMANCE=true: %v", skipConformanceVersions))
	}

	return errs
}

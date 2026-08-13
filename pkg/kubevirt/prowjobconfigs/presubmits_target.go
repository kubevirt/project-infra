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

package prowjobconfigs

import (
	"fmt"
	"regexp"
	"strconv"

	"sigs.k8s.io/prow/pkg/config"
)

type SigComputeJob struct {
	Name   string
	Target string
}

type SigNetworkJob struct {
	Name         string
	K8sVersion   string
	IsSmoke      bool
	SkipSmokeEnv string
}

var (
	sigComputeJobNameRegex = regexp.MustCompile(`^pull-kubevirt-e2e-k8s-(\d+\.\d+)-sig-compute(-(migrations|serial))?$`)
	sigNetworkJobNameRegex = regexp.MustCompile(`^pull-kubevirt-e2e-k8s-(\d+\.\d+)-sig-network(-smoke)?$`)
	k8sVersionRegex        = regexp.MustCompile(`k8s-(\d+)\.(\d+)`)
)

func CollectSigComputeJobs(jobConfig *config.JobConfig) []SigComputeJob {
	var jobs []SigComputeJob
	for _, job := range jobConfig.PresubmitsStatic[OrgAndRepoForJobConfig] {
		if !sigComputeJobNameRegex.MatchString(job.Name) {
			continue
		}
		for _, container := range job.Spec.Containers {
			for _, env := range container.Env {
				if env.Name == "TARGET" {
					jobs = append(jobs, SigComputeJob{Name: job.Name, Target: env.Value})
				}
			}
		}
	}
	return jobs
}

func CollectSigNetworkJobs(jobConfig *config.JobConfig) []SigNetworkJob {
	var jobs []SigNetworkJob
	for _, job := range jobConfig.PresubmitsStatic[OrgAndRepoForJobConfig] {
		matches := sigNetworkJobNameRegex.FindStringSubmatch(job.Name)
		if job.Optional || matches == nil {
			continue
		}
		jobs = append(jobs, SigNetworkJob{
			Name:         job.Name,
			K8sVersion:   matches[1],
			IsSmoke:      matches[2] == "-smoke",
			SkipSmokeEnv: GetEnvValue(job, "SKIP_SMOKE"),
		})
	}
	return jobs
}

func GetEnvValue(job config.Presubmit, envName string) string {
	for _, container := range job.Spec.Containers {
		for _, env := range container.Env {
			if env.Name == envName {
				return env.Value
			}
		}
	}
	return ""
}

func FindLatestK8sVersionFromJobs(jobs []SigComputeJob) (string, error) {
	var latestMajor, latestMinor int
	for _, job := range jobs {
		matches := k8sVersionRegex.FindStringSubmatch(job.Name)
		if matches == nil {
			continue
		}
		major, _ := strconv.Atoi(matches[1])
		minor, _ := strconv.Atoi(matches[2])
		if major > latestMajor || (major == latestMajor && minor > latestMinor) {
			latestMajor = major
			latestMinor = minor
		}
	}
	if latestMajor == 0 {
		return "", fmt.Errorf("could not determine latest k8s version from sig-compute job names")
	}
	return fmt.Sprintf("%d.%d", latestMajor, latestMinor), nil
}

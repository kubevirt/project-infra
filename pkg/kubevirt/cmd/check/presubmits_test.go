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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/prow/pkg/config"

	"kubevirt.io/project-infra/pkg/kubevirt/prowjobconfigs"
)

func TestPresubmits(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "check presubmits suite")
}

var _ = Describe("presubmit checks", func() {
	DescribeTable("validateSigNetworkSkipSmokeEnv",
		func(jobs []config.Presubmit, expectedErrs ...string) {
			errs := validateSigNetworkSkipSmokeEnv(&config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					prowjobconfigs.OrgAndRepoForJobConfig: jobs,
				},
			})

			if len(expectedErrs) == 0 {
				Expect(errs).To(BeEmpty())
				return
			}

			for _, expectedErr := range expectedErrs {
				Expect(errs).To(ContainElement(ContainSubstring(expectedErr)))
			}
		},
		Entry("smoke version matches env lane", []config.Presubmit{
			newSigNetworkJob("1.35", false, ""),
			newSigNetworkJob("1.36", false, "true"),
			newSigNetworkJob("1.36", true, ""),
		}),
		Entry("fails when env lane is missing", []config.Presubmit{
			newSigNetworkJob("1.35", false, ""),
			newSigNetworkJob("1.36", false, ""),
			newSigNetworkJob("1.36", true, ""),
		}, "no plain sig-network job has SKIP_SMOKE=true"),
		Entry("fails when smoke version mismatches env lane", []config.Presubmit{
			newSigNetworkJob("1.35", false, "true"),
			newSigNetworkJob("1.36", false, ""),
			newSigNetworkJob("1.36", true, ""),
		}, "sig-network-smoke runs on k8s-1.36, but SKIP_SMOKE=true is set on sig-network k8s-1.35"),
		Entry("fails when multiple env lanes exist", []config.Presubmit{
			newSigNetworkJob("1.35", false, "true"),
			newSigNetworkJob("1.36", false, "true"),
			newSigNetworkJob("1.36", true, ""),
		}, "multiple plain sig-network jobs have SKIP_SMOKE=true"),
		Entry("fails when env var is set to unexpected value", []config.Presubmit{
			newSigNetworkJob("1.36", false, "false"),
			newSigNetworkJob("1.36", true, ""),
		}, "job \"pull-kubevirt-e2e-k8s-1.36-sig-network\" has SKIP_SMOKE=\"false\", expected \"true\""),
		Entry("ignores optional extra smoke lane", []config.Presubmit{
			newSigNetworkJob("1.35", false, ""),
			newSigNetworkJob("1.36", false, "true"),
			newSigNetworkJob("1.36", true, ""),
			optionalPresubmit(newSigNetworkJob("1.37", true, "")),
		}),
		Entry("ignores optional SKIP_SMOKE on another version", []config.Presubmit{
			newSigNetworkJob("1.35", false, ""),
			newSigNetworkJob("1.36", false, "true"),
			newSigNetworkJob("1.36", true, ""),
			optionalPresubmit(newSigNetworkJob("1.37", false, "true")),
		}),
	)

	DescribeTable("validateSigComputeTargets",
		func(jobs []config.Presubmit, expectedErrs ...string) {
			errs := validateSigComputeTargets(&config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					prowjobconfigs.OrgAndRepoForJobConfig: jobs,
				},
			})

			if len(expectedErrs) == 0 {
				Expect(errs).To(BeEmpty())
				return
			}

			for _, expectedErr := range expectedErrs {
				Expect(errs).To(ContainElement(ContainSubstring(expectedErr)))
			}
		},
		Entry("accepts aligned latest jobs", []config.Presubmit{
			newSigComputeJob("1.35", false, "k8s-1.35-sig-compute-parallel"),
			newSigComputeJob("1.36", false, "k8s-1.36-sig-compute-parallel"),
			newSigComputeJob("1.36", true, "k8s-1.36-sig-compute-serial"),
		}),
		Entry("fails when latest parallel TARGET is wrong", []config.Presubmit{
			newSigComputeJob("1.36", false, "k8s-1.36-sig-compute"),
			newSigComputeJob("1.36", true, "k8s-1.36-sig-compute-serial"),
		}, "job \"pull-kubevirt-e2e-k8s-1.36-sig-compute\" has TARGET \"k8s-1.36-sig-compute\", expected \"k8s-1.36-sig-compute-parallel\""),
		Entry("fails when latest serial TARGET is wrong", []config.Presubmit{
			newSigComputeJob("1.36", false, "k8s-1.36-sig-compute-parallel"),
			newSigComputeJob("1.36", true, "k8s-1.36-sig-compute-parallel"),
		}, "job \"pull-kubevirt-e2e-k8s-1.36-sig-compute-serial\" has TARGET \"k8s-1.36-sig-compute-parallel\", expected \"k8s-1.36-sig-compute-serial\""),
	)
})

func newSigNetworkJob(version string, smoke bool, skipSmokeEnv string) config.Presubmit {
	name := "pull-kubevirt-e2e-k8s-" + version + "-sig-network"
	if smoke {
		name += "-smoke"
	}

	env := []v1.EnvVar{{
		Name:  "TARGET",
		Value: "k8s-" + version + "-sig-network",
	}}
	if smoke {
		env[0].Value += "-smoke"
	}
	if skipSmokeEnv != "" {
		env = append(env, v1.EnvVar{Name: "SKIP_SMOKE", Value: skipSmokeEnv})
	}

	return config.Presubmit{
		JobBase: config.JobBase{
			Name: name,
			Spec: &v1.PodSpec{
				Containers: []v1.Container{{
					Env: env,
				}},
			},
		},
	}
}

func optionalPresubmit(job config.Presubmit) config.Presubmit {
	job.Optional = true
	return job
}

func newSigComputeJob(version string, serial bool, target string) config.Presubmit {
	name := "pull-kubevirt-e2e-k8s-" + version + "-sig-compute"
	if serial {
		name += "-serial"
	}

	return config.Presubmit{
		JobBase: config.JobBase{
			Name: name,
			Spec: &v1.PodSpec{
				Containers: []v1.Container{{
					Env: []v1.EnvVar{{
						Name:  "TARGET",
						Value: target,
					}},
				}},
			},
		},
	}
}

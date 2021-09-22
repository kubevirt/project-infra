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
	"k8s.io/test-infra/prow/config"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/jobconfig"
	"kubevirt.io/project-infra/robots/pkg/querier"
	"testing"
)

func Test_ensureLatestJobsAreRequired(t *testing.T) {
	type args struct {
		jobConfigKubevirtPresubmits config.JobConfig
		release *querier.SemVer
	}
	tests := []struct {
		name string
		args args
		want latestJobsRequiredCheckResult
	}{
		{
			name: "not enough jobs for release found",
			args: args{
				jobConfigKubevirtPresubmits: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						jobconfig.OrgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", false, true, true),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-compute", false, true, true),
						},
					},
				},
				release: newMinorSemver("1", "37"),
			},
			want: NOT_ALL_JOBS_EXIST,
		},
		{
			name: "not all jobs for release are required",
			args: args{
				jobConfigKubevirtPresubmits: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						jobconfig.OrgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-compute", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-storage", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "operator", true, true, false),
						},
					},
				},
				release: newMinorSemver("1", "37"),
			},
			want: NOT_ALL_JOBS_ARE_REQUIRED,
		},
		{
			name: "all jobs for release are required",
			args: args{
				jobConfigKubevirtPresubmits: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						jobconfig.OrgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-compute", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-storage", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "operator", true, false, false),
						},
					},
				},
				release: newMinorSemver("1", "37"),
			},
			want: ALL_JOBS_ARE_REQUIRED,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, message := ensureLatestJobsAreRequired(tt.args.jobConfigKubevirtPresubmits, tt.args.release)
			if got != tt.want {
				t.Errorf("ensureLatestJobsAreRequired() = %v, want %v", got, tt.want)
			}
			t.Logf("message: %s", message)
		})
	}
}

func Test_ensureJobsExistForReleases(t *testing.T) {
	type args struct {
		jobConfigKubevirtPresubmits config.JobConfig
		requiredReleases            []*querier.SemVer
	}
	tests := []struct {
		name             string
		args             args
		wantAllJobsExist bool
	}{
		{
			name:             "jobs missing",
			args:             args{
				jobConfigKubevirtPresubmits: config.JobConfig{},
				requiredReleases: []*querier.SemVer{
					newMinorSemver("1", "37"),
					newMinorSemver("1", "42"),
				},
			},
			wantAllJobsExist: false,
		},
		{
			name:             "jobs exist",
			args:             args{
				jobConfigKubevirtPresubmits: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						jobconfig.OrgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-compute", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-storage", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "operator", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "42"), "sig-network", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "42"), "sig-compute", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "42"), "sig-storage", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "42"), "operator", true, false, false),
						},
					},
				},
				requiredReleases: []*querier.SemVer{
					newMinorSemver("1", "37"),
					newMinorSemver("1", "42"),
				},
			},
			wantAllJobsExist: true,
		},
	}
		for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAllJobsExist, gotMessage := ensureJobsExistForReleases(tt.args.jobConfigKubevirtPresubmits, tt.args.requiredReleases)
			if gotAllJobsExist != tt.wantAllJobsExist {
				t.Errorf("ensureJobsExistForReleases() gotAllJobsExist = %v, want %v", gotAllJobsExist, tt.wantAllJobsExist)
			}
			t.Logf("message: %s", gotMessage)
		})
	}
}

func newMinorSemver(major, minor string) *querier.SemVer {
	return &querier.SemVer{
		Major: major,
		Minor: minor,
		Patch: "0",
	}
}

func createPresubmitJobForRelease(semver *querier.SemVer, sigName string, alwaysRun, optional, skipReport bool) config.Presubmit {
	res := config.Presubmit{
		AlwaysRun: alwaysRun,
		Optional:  optional,
		JobBase: config.JobBase{
			Name: jobconfig.CreatePresubmitJobName(semver, sigName),
		},
		Reporter: config.Reporter{
			SkipReport: skipReport,
		},
	}
	return res
}

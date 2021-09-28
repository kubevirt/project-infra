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

package require

import (
	"reflect"
	"testing"

	"k8s.io/test-infra/prow/config"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/prowjobconfigs"
	"kubevirt.io/project-infra/robots/pkg/querier"
)

func TestUpdatePresubmitsAlwaysRunAndOptionalFields(t *testing.T) {
	type args struct {
		jobConfig           config.JobConfig
		latestReleaseSemver *querier.SemVer
	}
	tests := []struct {
		name             string
		args             args
		wantNewJobConfig config.JobConfig
		wantUpdated      bool
	}{
		{
			name: "no job exists",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						prowjobconfigs.OrgAndRepoForJobConfig: {},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					prowjobconfigs.OrgAndRepoForJobConfig: {},
				},
			},
			wantUpdated: false,
		},
		{
			name: "different k8s release job exists",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						prowjobconfigs.OrgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", false, true, true),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "42"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					prowjobconfigs.OrgAndRepoForJobConfig: {
						createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", false, true, true),
					},
				},
			},
			wantUpdated: false,
		},
		{
			name: "different sig job exists",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						prowjobconfigs.OrgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-other", false, true, true),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					prowjobconfigs.OrgAndRepoForJobConfig: {
						createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-other", false, true, true),
					},
				},
			},
			wantUpdated: false,
		},
		{
			name: "sig-network job exists, always_run = false",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						prowjobconfigs.OrgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", false, true, true),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					prowjobconfigs.OrgAndRepoForJobConfig: {
						createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, true, false),
					},
				},
			},
			wantUpdated: true,
		},
		{
			name: "sig-network job exists, always_run = true",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						prowjobconfigs.OrgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, true, false),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					prowjobconfigs.OrgAndRepoForJobConfig: {
						createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, false, false),
					},
				},
			},
			wantUpdated: true,
		},
		{
			name: "sig-network job exists, always_run = true, optional = false",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						prowjobconfigs.OrgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, false, false),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					prowjobconfigs.OrgAndRepoForJobConfig: {
						createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, false, false),
					},
				},
			},
			wantUpdated: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated := updatePresubmitsAlwaysRunAndOptionalFields(&tt.args.jobConfig, tt.args.latestReleaseSemver)
			if updated != tt.wantUpdated {
				t.Errorf("updatePresubmitsAlwaysRunAndOptionalFields() updated = %v, want %v", updated, tt.wantUpdated)
			}
			if !reflect.DeepEqual(tt.args.jobConfig, tt.wantNewJobConfig) {
				presubmit := tt.args.jobConfig.PresubmitsStatic["kubevirt/kubevirt"][0]
				t.Errorf("updatePresubmitsAlwaysRunAndOptionalFields() tt.args.jobConfig = %v, want %v\n\tAlwaysRun: %v, Optional: %v, SkipReport: %v, ", tt.args.jobConfig, tt.wantNewJobConfig, presubmit.AlwaysRun, presubmit.Optional, presubmit.SkipReport)
			}
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
			Name: prowjobconfigs.CreatePresubmitJobName(semver, sigName),
		},
		Reporter: config.Reporter{
			SkipReport: skipReport,
		},
	}
	return res
}

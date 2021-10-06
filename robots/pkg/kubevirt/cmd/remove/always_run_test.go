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
	"testing"

	"k8s.io/test-infra/prow/config"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/prowjobconfigs"
	"kubevirt.io/project-infra/robots/pkg/querier"
)

func Test_ensureSigJobsDoAlwaysRun(t *testing.T) {
	type args struct {
		jobConfigKubevirtPresubmits config.JobConfig
		release                     *querier.SemVer
	}
	tests := []struct {
		name        string
		args        args
		wantResult  latestJobsAlwaysRunCheckResult
		wantMessage string
	}{
		{
			name: "not enough jobs for release found",
			args: args{
				jobConfigKubevirtPresubmits: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						prowjobconfigs.OrgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", false, true, true),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-compute", false, true, true),
						},
					},
				},
				release: newMinorSemver("1", "37"),
			},
			wantResult: NOT_ALL_ALWAYS_RUN_JOBS_EXIST,
		},
		{
			name: "not all jobs for release do always run",
			args: args{
				jobConfigKubevirtPresubmits: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						prowjobconfigs.OrgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-compute", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-storage", false, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "operator", true, true, false),
						},
					},
				},
				release: newMinorSemver("1", "37"),
			},
			wantResult: NOT_ALL_JOBS_DO_ALWAYS_RUN,
		},
		{
			name: "all jobs for release do always run",
			args: args{
				jobConfigKubevirtPresubmits: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						prowjobconfigs.OrgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-compute", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-storage", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "operator", true, false, false),
						},
					},
				},
				release: newMinorSemver("1", "37"),
			},
			wantResult: ALL_JOBS_DO_ALWAYS_RUN,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResult, gotMessage := ensureSigJobsDoAlwaysRun(tt.args.jobConfigKubevirtPresubmits, tt.args.release)
			if gotResult != tt.wantResult {
				t.Errorf("ensureSigJobsDoAlwaysRun() gotResult = %v, want %v", gotResult, tt.wantResult)
			}
			t.Logf("ensureSigJobsDoAlwaysRun() gotMessage = %v", gotMessage)
		})
	}
}

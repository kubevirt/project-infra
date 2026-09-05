package main

import (
	"reflect"
	"testing"

	"kubevirt.io/project-infra/pkg/querier"
	"sigs.k8s.io/prow/pkg/config"
)

func TestDeletePresubmitJobForRelease(t *testing.T) {
	type args struct {
		jobConfig           *config.JobConfig
		targetReleaseSemver *querier.SemVer
		extraArchs          []string
	}
	tests := []struct {
		name             string
		args             args
		wantNewJobConfig *config.JobConfig
		wantUpdated      bool
	}{
		{
			name: "no job exists",
			args: args{
				jobConfig: &config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						OrgAndRepoForJobConfig: {},
					},
				},
				targetReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: &config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					OrgAndRepoForJobConfig: {},
				},
			},
			wantUpdated: false,
		},
		{
			name: "different job exists",
			args: args{
				jobConfig: &config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						OrgAndRepoForJobConfig: {
							{
								JobBase: config.JobBase{
									Name: createKubevirtciPresubmitJobName(newMinorSemver("1", "42")),
								},
							},
						},
					},
				},
				targetReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: &config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					OrgAndRepoForJobConfig: {
						{
							JobBase: config.JobBase{
								Name: createKubevirtciPresubmitJobName(newMinorSemver("1", "42")),
							},
						},
					},
				},
			},
			wantUpdated: false,
		},
		{
			name: "same job exists",
			args: args{
				jobConfig: &config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						OrgAndRepoForJobConfig: {
							{
								JobBase: config.JobBase{
									Name: createKubevirtciPresubmitJobName(newMinorSemver("1", "37")),
								},
							},
						},
					},
				},
				targetReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: &config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					OrgAndRepoForJobConfig: nil,
				},
			},
			wantUpdated: true,
		},
		{
			name: "same job exists but different patch version",
			args: args{
				jobConfig: &config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						OrgAndRepoForJobConfig: {
							{
								JobBase: config.JobBase{
									Name: createKubevirtciPresubmitJobName(newSemver("1", "37", "0")),
								},
							},
						},
					},
				},
				targetReleaseSemver: newSemver("1", "37", "1"),
			},
			wantNewJobConfig: &config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					OrgAndRepoForJobConfig: nil,
				},
			},
			wantUpdated: true,
		},
		{
			name: "only amd64 deleted when no extra archs",
			args: args{
				jobConfig: &config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						OrgAndRepoForJobConfig: {
							{
								JobBase: config.JobBase{
									Name: createKubevirtciPresubmitJobName(newMinorSemver("1", "37")),
								},
							},
							{
								JobBase: config.JobBase{
									Name: createKubevirtciPresubmitJobNameArch(newMinorSemver("1", "37"), "s390x"),
								},
							},
						},
					},
				},
				targetReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: &config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					OrgAndRepoForJobConfig: {
						{
							JobBase: config.JobBase{
								Name: createKubevirtciPresubmitJobNameArch(newMinorSemver("1", "37"), "s390x"),
							},
						},
					},
				},
			},
			wantUpdated: true,
		},
		{
			name: "both amd64 and arch jobs deleted with extra archs",
			args: args{
				jobConfig: &config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						OrgAndRepoForJobConfig: {
							{
								JobBase: config.JobBase{
									Name: createKubevirtciPresubmitJobName(newMinorSemver("1", "37")),
								},
							},
							{
								JobBase: config.JobBase{
									Name: createKubevirtciPresubmitJobNameArch(newMinorSemver("1", "37"), "s390x"),
								},
							},
							{
								JobBase: config.JobBase{
									Name: createKubevirtciPresubmitJobName(newMinorSemver("1", "42")),
								},
							},
						},
					},
				},
				targetReleaseSemver: newMinorSemver("1", "37"),
				extraArchs:          []string{"s390x"},
			},
			wantNewJobConfig: &config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					OrgAndRepoForJobConfig: {
						{
							JobBase: config.JobBase{
								Name: createKubevirtciPresubmitJobName(newMinorSemver("1", "42")),
							},
						},
					},
				},
			},
			wantUpdated: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUpdated := deletePresubmitJobForRelease(tt.args.jobConfig, tt.args.targetReleaseSemver, tt.args.extraArchs)
			if gotUpdated != tt.wantUpdated {
				t.Errorf("deletePresubmitJobForRelease() gotUpdated = %v, want %v", gotUpdated, tt.wantUpdated)
			}
			if tt.wantUpdated && !reflect.DeepEqual(tt.args.jobConfig, tt.wantNewJobConfig) {
				t.Errorf("deletePresubmitJobForRelease() gotNewJobConfig = %v, want %v", tt.args.jobConfig, tt.wantNewJobConfig)
			}
		})
	}
}

func newSemver(major, minor, patch string) *querier.SemVer {
	return &querier.SemVer{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
}

func newMinorSemver(major, minor string) *querier.SemVer {
	return &querier.SemVer{
		Major: major,
		Minor: minor,
		Patch: "0",
	}
}

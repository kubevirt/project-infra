package main

import (
	"reflect"
	"testing"

	v1 "k8s.io/api/core/v1"
	"kubevirt.io/project-infra/pkg/querier"
	"sigs.k8s.io/prow/pkg/config"
)

func TestCreateBumpedJobForRelease(t *testing.T) {
	type args struct {
		expectedJob config.Presubmit
		semver      *querier.SemVer
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "job 1.37 is created",
			args: args{
				expectedJob: config.Presubmit{
					JobBase: config.JobBase{
						Name: "check-provision-k8s-1.37",
						Spec: &v1.PodSpec{
							Containers: []v1.Container{
								{
									Command: []string{
										"/usr/local/bin/runner.sh",
										"/bin/sh",
										"-c",
										"cd cluster-provision/k8s/1.37 && KUBEVIRT_PSA='true' ../provision.sh",
									},
								},
							},
						},
					},
					AlwaysRun: false,
					Optional:  true,
				},
				semver: newMinorSemver("1", "37"),
			},
		},
		{
			name: "job 1.42 is created",
			args: args{
				expectedJob: config.Presubmit{
					JobBase: config.JobBase{
						Name: "check-provision-k8s-1.42",
						Spec: &v1.PodSpec{
							Containers: []v1.Container{
								{
									Command: []string{
										"/usr/local/bin/runner.sh",
										"/bin/sh",
										"-c",
										"cd cluster-provision/k8s/1.42 && KUBEVIRT_PSA='true' ../provision.sh",
									},
								},
							},
						},
					},
					AlwaysRun: false,
					Optional:  true,
				},
				semver: newMinorSemver("1", "42"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bumpedJobForRelease := CreatePresubmitJobForRelease(tt.args.semver)
			if bumpedJobForRelease.Name != tt.args.expectedJob.Name {
				t.Errorf("job name differs, got = %v,\n want %v", bumpedJobForRelease.Name, tt.args.expectedJob.Name)
			}
			if bumpedJobForRelease.AlwaysRun != tt.args.expectedJob.AlwaysRun {
				t.Errorf("job AlwaysRun differs, got = %v,\n want %v", bumpedJobForRelease.AlwaysRun, tt.args.expectedJob.AlwaysRun)
			}
			if bumpedJobForRelease.Optional != tt.args.expectedJob.Optional {
				t.Errorf("job Optional differs, got = %v,\n want %v", bumpedJobForRelease.Optional, tt.args.expectedJob.Optional)
			}
			if !reflect.DeepEqual(bumpedJobForRelease.Spec.Containers[0].Command, tt.args.expectedJob.Spec.Containers[0].Command) {
				t.Errorf("job spec differs, got = %v,\n want %v", bumpedJobForRelease, tt.args.expectedJob)
			}
		})
	}
}

func TestAddNewPresubmitIfNotExists(t *testing.T) {
	type args struct {
		jobConfig           config.JobConfig
		latestReleaseSemver *querier.SemVer
	}
	tests := []struct {
		name             string
		args             args
		wantNewJobConfig config.JobConfig
		wantJobExists    bool
	}{
		{
			name: "no job exists",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						OrgAndRepoForJobConfig: {},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					OrgAndRepoForJobConfig: {
						CreatePresubmitJobForRelease(newMinorSemver("1", "37")),
					},
				},
			},
			wantJobExists: false,
		},
		{
			name: "different job exists",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						OrgAndRepoForJobConfig: {
							CreatePresubmitJobForRelease(newMinorSemver("1", "37")),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "42"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					OrgAndRepoForJobConfig: {
						CreatePresubmitJobForRelease(newMinorSemver("1", "37")),
						CreatePresubmitJobForRelease(newMinorSemver("1", "42")),
					},
				},
			},
			wantJobExists: false,
		},
		{
			name: "same job exists",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						OrgAndRepoForJobConfig: {
							CreatePresubmitJobForRelease(newMinorSemver("1", "37")),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					OrgAndRepoForJobConfig: {
						CreatePresubmitJobForRelease(newMinorSemver("1", "37")),
					},
				},
			},
			wantJobExists: true,
		},
		{
			name: "same job exists but different patch version",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						OrgAndRepoForJobConfig: {
							CreatePresubmitJobForRelease(newSemver("1", "37", "0")),
						},
					},
				},
				latestReleaseSemver: newSemver("1", "37", "1"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					OrgAndRepoForJobConfig: {
						CreatePresubmitJobForRelease(newSemver("1", "37", "0")),
					},
				},
			},
			wantJobExists: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNewJobConfig, gotJobExists := AddNewPresubmitIfNotExists(tt.args.jobConfig, tt.args.latestReleaseSemver)
			if !reflect.DeepEqual(gotNewJobConfig, tt.wantNewJobConfig) {
				t.Errorf("AddNewPresubmitIfNotExists() gotNewJobConfig = %v, want %v", gotNewJobConfig, tt.wantNewJobConfig)
			}
			if gotJobExists != tt.wantJobExists {
				t.Errorf("AddNewPresubmitIfNotExists() gotJobExists = %v, want %v", gotJobExists, tt.wantJobExists)
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

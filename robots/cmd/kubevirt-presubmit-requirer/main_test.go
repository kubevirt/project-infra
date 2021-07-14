package main

import (
	"k8s.io/test-infra/prow/config"
	"kubevirt.io/project-infra/robots/pkg/querier"
	"reflect"
	"testing"
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
		wantUpdated    	 bool
	}{
		{
			name: "no job exists",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						orgAndRepoForJobConfig: {},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					orgAndRepoForJobConfig: {},
				},
			},
			wantUpdated: false,
		},
		{
			name: "different k8s release job exists",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						orgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", false, "", "", true, true),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "42"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					orgAndRepoForJobConfig: {
						createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", false, "", "", true, true),
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
						orgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-other", false, "", "", true, true),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					orgAndRepoForJobConfig: {
						createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-other", false, "", "", true, true),
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
						orgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", false, "", "", true, true),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					orgAndRepoForJobConfig: {
						createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", false, "", "^docs/|.md$|^LICENSE$", true, false),
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
						orgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, "", "", true, false),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					orgAndRepoForJobConfig: {
						createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, "", "", false, false),
					},
				},
			},
			wantUpdated: true,
		},
		{
			name: "sig-network job exists, always_run = false, run_if_changed set",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						orgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", false, "src/.*", "", true, false),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					orgAndRepoForJobConfig: {
						createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, "", "", false, false),
					},
				},
			},
			wantUpdated: true,
		},
		{
			name: "sig-network job exists, always_run = false, skip_if_only_changed set",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						orgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", false, "", "docs/.*", true, false),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					orgAndRepoForJobConfig: {
						createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, "", "", false, false),
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
						orgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, "", "", false, false),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					orgAndRepoForJobConfig: {
						createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, "", "", false, false),
					},
				},
			},
			wantUpdated: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updated := UpdatePresubmitsAlwaysRunAndOptionalFields(&tt.args.jobConfig, tt.args.latestReleaseSemver)
			if updated != tt.wantUpdated {
				t.Errorf("UpdatePresubmitsAlwaysRunAndOptionalFields() updated = %v, want %v", updated, tt.wantUpdated)
			}
			if !reflect.DeepEqual(tt.args.jobConfig, tt.wantNewJobConfig) {
				presubmit := tt.args.jobConfig.PresubmitsStatic["kubevirt/kubevirt"][0]
				t.Errorf("UpdatePresubmitsAlwaysRunAndOptionalFields() tt.args.jobConfig = %v, want %v\n\tAlwaysRun: %v, RunIfChanged: %v, SkipIfOnlyChanged: %v, Optional: %v, SkipReport: %v, ", tt.args.jobConfig, tt.wantNewJobConfig, presubmit.AlwaysRun, presubmit.RegexpChangeMatcher.RunIfChanged, presubmit.RegexpChangeMatcher.SkipIfOnlyChanged, presubmit.Optional, presubmit.SkipReport)
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

func createPresubmitJobForRelease(semver *querier.SemVer, sigName string, alwaysRun bool, runIfChanged, skipIfOnlyChanged string, optional, skipReport bool) config.Presubmit {
	return config.Presubmit{
		AlwaysRun: alwaysRun,
		Optional:  optional,
		JobBase: config.JobBase{
			Name: createPresubmitJobName(semver, sigName),
		},
		RegexpChangeMatcher: config.RegexpChangeMatcher{
			RunIfChanged: runIfChanged,
			SkipIfOnlyChanged: skipIfOnlyChanged,
		},
		Reporter: config.Reporter{
			SkipReport: skipReport,
		},
	}
}

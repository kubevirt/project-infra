package main

import (
	"reflect"
	"testing"
	"time"

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
										"/usr/local/bin/entrypoint.sh",
										"/bin/sh",
										"-c",
										"cd cluster-provision/k8s/1.37 && ../provision.sh",
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
										"/usr/local/bin/entrypoint.sh",
										"/bin/sh",
										"-c",
										"cd cluster-provision/k8s/1.42 && ../provision.sh",
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

func TestCreateBumpedJobForReleaseArch(t *testing.T) {
	semver := newMinorSemver("1", "37")
	archCfg := knownArchConfigs["s390x"]
	job := CreatePresubmitJobForReleaseArch(semver, "s390x", archCfg)

	if job.Name != "check-provision-k8s-1.37-s390x" {
		t.Errorf("job name = %v, want check-provision-k8s-1.37-s390x", job.Name)
	}
	if job.Cluster != "prow-s390x-workloads" {
		t.Errorf("cluster = %v, want prow-s390x-workloads", job.Cluster)
	}
	if job.DecorationConfig.Timeout.Duration != 4*time.Hour {
		t.Errorf("timeout = %v, want 4h", job.DecorationConfig.Timeout.Duration)
	}
	if job.AlwaysRun != false {
		t.Errorf("AlwaysRun = %v, want false", job.AlwaysRun)
	}
	if _, ok := job.Labels["preset-docker-mirror-proxy"]; ok {
		t.Error("arch job should not have preset-docker-mirror-proxy label")
	}
	if job.Labels["preset-kubevirtci-check-provision-env"] != "true" {
		t.Error("arch job should have preset-kubevirtci-check-provision-env label")
	}
	if job.Labels["preset-podman-in-container-enabled"] != "true" {
		t.Error("arch job should have preset-podman-in-container-enabled label")
	}
	if job.Spec.NodeSelector != nil {
		t.Errorf("arch job should have nil NodeSelector, got %v", job.Spec.NodeSelector)
	}

	envMap := make(map[string]string)
	for _, env := range job.Spec.Containers[0].Env {
		envMap[env.Name] = env.Value
	}
	if envMap["SLIM"] != "true" {
		t.Error("arch job should have SLIM=true env var")
	}
	if envMap["RUN_KUBEVIRT_CONFORMANCE"] != "false" {
		t.Error("arch job should have RUN_KUBEVIRT_CONFORMANCE=false env var")
	}
	if envMap["GO_MOD_PATH"] != "cluster-provision/gocli/go.mod" {
		t.Error("arch job should have GO_MOD_PATH env var")
	}

	expectedCmd := []string{
		"/usr/local/bin/entrypoint.sh",
		"/bin/sh",
		"-c",
		"cd cluster-provision/k8s/1.37 && ../provision.sh",
	}
	if !reflect.DeepEqual(job.Spec.Containers[0].Command, expectedCmd) {
		t.Errorf("command = %v, want %v", job.Spec.Containers[0].Command, expectedCmd)
	}
}

func TestAddNewPresubmitIfNotExists(t *testing.T) {
	type args struct {
		jobConfig           config.JobConfig
		latestReleaseSemver *querier.SemVer
		extraArchs          []string
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
		{
			name: "no job exists with extra archs",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						OrgAndRepoForJobConfig: {},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
				extraArchs:          []string{"s390x"},
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					OrgAndRepoForJobConfig: {
						CreatePresubmitJobForRelease(newMinorSemver("1", "37")),
						CreatePresubmitJobForReleaseArch(newMinorSemver("1", "37"), "s390x", knownArchConfigs["s390x"]),
					},
				},
			},
			wantJobExists: false,
		},
		{
			name: "amd64 exists but arch job missing",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						OrgAndRepoForJobConfig: {
							CreatePresubmitJobForRelease(newMinorSemver("1", "37")),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
				extraArchs:          []string{"s390x"},
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					OrgAndRepoForJobConfig: {
						CreatePresubmitJobForRelease(newMinorSemver("1", "37")),
						CreatePresubmitJobForReleaseArch(newMinorSemver("1", "37"), "s390x", knownArchConfigs["s390x"]),
					},
				},
			},
			wantJobExists: false,
		},
		{
			name: "arch job exists but amd64 missing",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						OrgAndRepoForJobConfig: {
							CreatePresubmitJobForReleaseArch(newMinorSemver("1", "37"), "s390x", knownArchConfigs["s390x"]),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
				extraArchs:          []string{"s390x"},
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					OrgAndRepoForJobConfig: {
						CreatePresubmitJobForReleaseArch(newMinorSemver("1", "37"), "s390x", knownArchConfigs["s390x"]),
						CreatePresubmitJobForRelease(newMinorSemver("1", "37")),
					},
				},
			},
			wantJobExists: false,
		},
		{
			name: "both amd64 and arch jobs exist",
			args: args{
				jobConfig: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						OrgAndRepoForJobConfig: {
							CreatePresubmitJobForRelease(newMinorSemver("1", "37")),
							CreatePresubmitJobForReleaseArch(newMinorSemver("1", "37"), "s390x", knownArchConfigs["s390x"]),
						},
					},
				},
				latestReleaseSemver: newMinorSemver("1", "37"),
				extraArchs:          []string{"s390x"},
			},
			wantNewJobConfig: config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					OrgAndRepoForJobConfig: {
						CreatePresubmitJobForRelease(newMinorSemver("1", "37")),
						CreatePresubmitJobForReleaseArch(newMinorSemver("1", "37"), "s390x", knownArchConfigs["s390x"]),
					},
				},
			},
			wantJobExists: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNewJobConfig, gotJobExists := AddNewPresubmitIfNotExists(tt.args.jobConfig, tt.args.latestReleaseSemver, tt.args.extraArchs)
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

package main

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/test-infra/prow/config"
	"kubevirt.io/project-infra/robots/pkg/querier"
	"reflect"
	"testing"
)

func TestCreateBumpedJobForRelease(t *testing.T) {
	type args struct {
		expectedJob config.Presubmit
		expectedErr bool
		semver      *querier.SemVer
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "job is created",
			args: args{
				expectedJob: config.Presubmit{
					JobBase: config.JobBase{
						Name: "check-provision-k8s-1.21",
						Spec: &v1.PodSpec{
							Containers: []v1.Container{
								{
									Command: []string{
										"/usr/local/bin/runner.sh",
										"/bin/sh",
										"-c",
										"cd cluster-provision/k8s/1.21 && ../provision.sh",
									},
								},
							},
						},
					},
					AlwaysRun: false,
					Optional:  true,
				},
				semver: newSemver("1", "21", "0"),
			},
		},
		// TODO
		//{
		//	name: "bump failure",
		//	args: args{
		//		job: config.Presubmit{
		//			JobBase: config.JobBase{
		//				Name:            "check-provision-k8s-1.20",
		//				Labels:          nil,
		//				MaxConcurrency:  0,
		//				Agent:           "",
		//				Cluster:         "",
		//				Namespace:       nil,
		//				ErrorOnEviction: false,
		//				SourcePath:      "",
		//				Spec:            &v1.PodSpec{
		//					Containers:                    []v1.Container{
		//						{
		//							Command: []string{
		//								"/usr/local/bin/runner.sh",
		//								"/bin/sh",
		//								"-c",
		//								"cd cluster-provision/k8s/1.20 && ../provision.sh",
		//															},
		//														},
		//					},
		//				},
		//				PipelineRunSpec: nil,
		//				Annotations:     nil,
		//				ReporterConfig:  nil,
		//				RerunAuthConfig: nil,
		//				Hidden:          false,
		//				UtilityConfig:   config.UtilityConfig{},
		//			},
		//			AlwaysRun:           false,
		//			Optional:            false,
		//			Trigger:             "",
		//			RerunCommand:        "",
		//			Brancher:            config.Brancher{},
		//			RegexpChangeMatcher: config.RegexpChangeMatcher{},
		//			Reporter:            config.Reporter{},
		//			JenkinsSpec:         nil,
		//		},
		//		semver: newSemver("1", "20", "0"),
		//	},
		//},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bumpedJobForRelease, err := CreatePresubmitJobForRelease(tt.args.semver)
			if (err != nil) != tt.args.expectedErr {
				t.Errorf("error expected: %t, err: %v", tt.args.expectedErr, err)
			}
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

func newSemver(major, minor, patch string) *querier.SemVer {
	return &querier.SemVer{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
}

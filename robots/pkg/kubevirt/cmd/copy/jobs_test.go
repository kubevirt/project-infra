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

package copy

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strings"
	"testing"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/release"

	"sigs.k8s.io/yaml"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/prowjobconfigs"

	"github.com/go-test/deep"
	"github.com/google/go-github/github"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/config"

	"kubevirt.io/project-infra/robots/pkg/querier"
)

func Test_getSourceAndTargetRelease(t *testing.T) {
	type args struct {
		releases []*github.RepositoryRelease
	}
	tests := []struct {
		name              string
		args              args
		wantTargetRelease *querier.SemVer
		wantSourceRelease *querier.SemVer
		wantErr           error
	}{
		{
			name: "has one patch Release for latest",
			args: args{
				releases: []*github.RepositoryRelease{
					release.Release("v1.22.0"),
					release.Release("v1.21.3"),
				},
			},
			wantTargetRelease: &querier.SemVer{
				Major: "1",
				Minor: "22",
				Patch: "0",
			},
			wantSourceRelease: &querier.SemVer{
				Major: "1",
				Minor: "21",
				Patch: "3",
			},
			wantErr: nil,
		},
		{
			name: "has two patch releases for latest",
			args: args{
				releases: []*github.RepositoryRelease{
					release.Release("v1.22.1"),
					release.Release("v1.22.0"),
					release.Release("v1.21.3"),
				},
			},
			wantTargetRelease: &querier.SemVer{
				Major: "1",
				Minor: "22",
				Patch: "1",
			},
			wantSourceRelease: &querier.SemVer{
				Major: "1",
				Minor: "21",
				Patch: "3",
			},
			wantErr: nil,
		},
		{
			name: "has one Release only, should err",
			args: args{
				releases: []*github.RepositoryRelease{
					release.Release("v1.22.1"),
				},
			},
			wantTargetRelease: nil,
			wantSourceRelease: nil,
			wantErr:           fmt.Errorf("less than two releases"),
		},
		{
			name: "has two major same releases",
			args: args{
				releases: []*github.RepositoryRelease{
					release.Release("v1.22.1"),
					release.Release("v1.22.0"),
				},
			},
			wantTargetRelease: &querier.SemVer{
				Major: "1",
				Minor: "22",
				Patch: "1",
			},
			wantSourceRelease: nil,
			wantErr:           fmt.Errorf("no source Release found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTargetRelease, gotSourceRelease, gotErr := getSourceAndTargetRelease(tt.args.releases)
			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("getSourceAndTargetRelease() got = %v, want %v", gotErr, tt.wantErr)
			}
			if !reflect.DeepEqual(gotTargetRelease, tt.wantTargetRelease) {
				t.Errorf("getSourceAndTargetRelease() got = %v, want %v", gotTargetRelease, tt.wantTargetRelease)
			}
			if !reflect.DeepEqual(gotSourceRelease, tt.wantSourceRelease) {
				t.Errorf("getSourceAndTargetRelease() got1 = %v, want %v", gotSourceRelease, tt.wantSourceRelease)
			}
		})
	}
}

func Test_copyPeriodicJobsForNewProvider(t *testing.T) {
	type args struct {
		jobConfig                   *config.JobConfig
		targetProviderReleaseSemver *querier.SemVer
		sourceProviderReleaseSemver *querier.SemVer
	}
	tests := []struct {
		name                                 string
		args                                 args
		wantUpdated                          bool
		wantJobConfig                        *config.JobConfig
		wantJobStatesToReportInSerialization bool
	}{
		/*
			checks that in the case of explicitly leaving the job_states_to_report empty, which implies that we do not
			want to report in any case, this empty config is preserved in the resulting modified version that has been
			written to storage.
		*/
		{
			name: "reporterconfig with empty job states slice is preserved even with no job state to report",
			args: args{
				jobConfig: &config.JobConfig{
					Periodics: []config.Periodic{
						{
							JobBase: config.JobBase{
								Labels: map[string]string{},
								ReporterConfig: &v1.ReporterConfig{
									Slack: &v1.SlackReporterConfig{
										JobStatesToReport: []v1.ProwJobState{},
									},
								},
								Name: prowjobconfigs.CreatePeriodicJobName(semver("1", "21", "0"), "sig-network"),
								Spec: &corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Env: []corev1.EnvVar{},
										},
									},
								},
							},
							Interval: "",
							Cron:     "0 1,9,17 * * *",
							Tags:     nil,
						},
					},
				},
				targetProviderReleaseSemver: semver("1", "22", "0"),
				sourceProviderReleaseSemver: semver("1", "21", "0"),
			},
			wantUpdated: true,
			wantJobConfig: &config.JobConfig{
				Periodics: []config.Periodic{
					{
						JobBase: config.JobBase{
							Labels: map[string]string{},
							Name:   prowjobconfigs.CreatePeriodicJobName(semver("1", "21", "0"), "sig-network"),
							ReporterConfig: &v1.ReporterConfig{
								Slack: &v1.SlackReporterConfig{
									JobStatesToReport: []v1.ProwJobState{},
								},
							},
							Spec: &corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Env: []corev1.EnvVar{},
									},
								},
							},
						},
						Interval: "",
						Cron:     "0 1,9,17 * * *",
						Tags:     nil,
					},
					{
						JobBase: config.JobBase{
							Annotations: map[string]string{},
							Labels:      map[string]string{},
							Name:        prowjobconfigs.CreatePeriodicJobName(semver("1", "22", "0"), "sig-network"),
							ReporterConfig: &v1.ReporterConfig{
								Slack: &v1.SlackReporterConfig{
									JobStatesToReport: []v1.ProwJobState{},
								},
							},
							Spec: &corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Env: []corev1.EnvVar{},
									},
								},
							},
						},
						Interval: "",
						Cron:     "10 2,10,18 * * *",
						Tags:     nil,
					},
				},
			},
			wantJobStatesToReportInSerialization: true,
		},
		{
			name: "extra_refs field exists for new provider job",
			args: args{
				jobConfig: &config.JobConfig{
					Periodics: []config.Periodic{
						{
							JobBase: config.JobBase{
								Labels: map[string]string{},
								Name:   prowjobconfigs.CreatePeriodicJobName(semver("1", "21", "0"), "sig-network"),
								Spec: &corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Env: []corev1.EnvVar{},
										},
									},
								},
								UtilityConfig: config.UtilityConfig{
									ExtraRefs: []v1.Refs{
										{
											Org:     "kubevirt",
											Repo:    "kubevirt",
											BaseRef: "main",
										},
									},
								},
							},
							Interval: "",
							Cron:     "0 1,9,17 * * *",
							Tags:     nil,
						},
					},
				},
				targetProviderReleaseSemver: semver("1", "22", "0"),
				sourceProviderReleaseSemver: semver("1", "21", "0"),
			},
			wantUpdated: true,
			wantJobConfig: &config.JobConfig{
				Periodics: []config.Periodic{
					{
						JobBase: config.JobBase{
							Labels: map[string]string{},
							Name:   prowjobconfigs.CreatePeriodicJobName(semver("1", "21", "0"), "sig-network"),
							Spec: &corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Env: []corev1.EnvVar{},
									},
								},
							},
							UtilityConfig: config.UtilityConfig{
								ExtraRefs: []v1.Refs{
									{
										Org:     "kubevirt",
										Repo:    "kubevirt",
										BaseRef: "main",
									},
								},
							},
						},
						Interval: "",
						Cron:     "0 1,9,17 * * *",
						Tags:     nil,
					},
					{
						JobBase: config.JobBase{
							Annotations: map[string]string{},
							Labels:      map[string]string{},
							Name:        prowjobconfigs.CreatePeriodicJobName(semver("1", "22", "0"), "sig-network"),
							Spec: &corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Env: []corev1.EnvVar{},
									},
								},
							},
							UtilityConfig: config.UtilityConfig{
								ExtraRefs: []v1.Refs{
									{
										Org:     "kubevirt",
										Repo:    "kubevirt",
										BaseRef: "main",
									},
								},
							},
						},
						Interval: "",
						Cron:     "10 2,10,18 * * *",
						Tags:     nil,
					},
				},
			},
			wantJobStatesToReportInSerialization: false,
		},
		{
			name: "annotations and labels are copied",
			args: args{
				jobConfig: &config.JobConfig{
					Periodics: []config.Periodic{
						{
							JobBase: config.JobBase{
								Annotations: map[string]string{
									"your-annotation": "value-goes-here",
								},
								Labels: map[string]string{
									"your-label": "value-goes-here",
								},
								Name: prowjobconfigs.CreatePeriodicJobName(semver("1", "21", "0"), "sig-network"),
								Spec: &corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Env: []corev1.EnvVar{},
										},
									},
								},
								UtilityConfig: config.UtilityConfig{
									ExtraRefs: []v1.Refs{
										{
											Org:     "kubevirt",
											Repo:    "kubevirt",
											BaseRef: "main",
										},
									},
								},
							},
							Interval: "",
							Cron:     "0 1,9,17 * * *",
							Tags:     nil,
						},
					},
				},
				targetProviderReleaseSemver: semver("1", "22", "0"),
				sourceProviderReleaseSemver: semver("1", "21", "0"),
			},
			wantUpdated: true,
			wantJobConfig: &config.JobConfig{
				Periodics: []config.Periodic{
					{
						JobBase: config.JobBase{
							Annotations: map[string]string{
								"your-annotation": "value-goes-here",
							},
							Labels: map[string]string{
								"your-label": "value-goes-here",
							},
							Name: prowjobconfigs.CreatePeriodicJobName(semver("1", "21", "0"), "sig-network"),
							Spec: &corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Env: []corev1.EnvVar{},
									},
								},
							},
							UtilityConfig: config.UtilityConfig{
								ExtraRefs: []v1.Refs{
									{
										Org:     "kubevirt",
										Repo:    "kubevirt",
										BaseRef: "main",
									},
								},
							},
						},
						Interval: "",
						Cron:     "0 1,9,17 * * *",
						Tags:     nil,
					},
					{
						JobBase: config.JobBase{
							Annotations: map[string]string{
								"your-annotation": "value-goes-here",
							},
							Labels: map[string]string{
								"your-label": "value-goes-here",
							},
							Name: prowjobconfigs.CreatePeriodicJobName(semver("1", "22", "0"), "sig-network"),
							Spec: &corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Env: []corev1.EnvVar{},
									},
								},
							},
							UtilityConfig: config.UtilityConfig{
								ExtraRefs: []v1.Refs{
									{
										Org:     "kubevirt",
										Repo:    "kubevirt",
										BaseRef: "main",
									},
								},
							},
						},
						Interval: "",
						Cron:     "10 2,10,18 * * *",
						Tags:     nil,
					},
				},
			},
			wantJobStatesToReportInSerialization: false,
		},
		{
			name: "have multiple containers, check TARGET env var",
			args: args{
				jobConfig: &config.JobConfig{
					Periodics: []config.Periodic{
						{
							JobBase: config.JobBase{
								Labels: map[string]string{},
								Name:   prowjobconfigs.CreatePeriodicJobName(semver("1", "21", "0"), "sig-network"),
								Spec: &corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Env: []corev1.EnvVar{
												{
													Name:  "Test1",
													Value: "Blah1",
												},
											},
										},
										{
											Env: []corev1.EnvVar{
												{
													Name:  "TARGET",
													Value: "k8s-1.21-sig-network",
												},
											},
										},
									},
								},
							},
							Interval: "",
							Cron:     "0 1,9,17 * * *",
							Tags:     nil,
						},
					},
				},
				targetProviderReleaseSemver: semver("1", "22", "0"),
				sourceProviderReleaseSemver: semver("1", "21", "0"),
			},
			wantUpdated: true,
			wantJobConfig: &config.JobConfig{
				Periodics: []config.Periodic{
					{
						JobBase: config.JobBase{
							Labels: map[string]string{},
							Name:   prowjobconfigs.CreatePeriodicJobName(semver("1", "21", "0"), "sig-network"),
							Spec: &corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Env: []corev1.EnvVar{
											{
												Name:  "Test1",
												Value: "Blah1",
											},
										},
									},
									{
										Env: []corev1.EnvVar{
											{
												Name:  "TARGET",
												Value: "k8s-1.21-sig-network",
											},
										},
									},
								},
							},
						},
						Interval: "",
						Cron:     "0 1,9,17 * * *",
						Tags:     nil,
					},
					{
						JobBase: config.JobBase{
							Annotations: map[string]string{},
							Labels:      map[string]string{},
							Name:        prowjobconfigs.CreatePeriodicJobName(semver("1", "22", "0"), "sig-network"),
							Spec: &corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Env: []corev1.EnvVar{
											{
												Name:  "Test1",
												Value: "Blah1",
											},
										},
									},
									{
										Env: []corev1.EnvVar{
											{
												Name:  "TARGET",
												Value: "k8s-1.22-sig-network",
											},
										},
									},
								},
							},
						},
						Interval: "",
						Cron:     "10 2,10,18 * * *",
						Tags:     nil,
					},
				},
			},
			wantJobStatesToReportInSerialization: false,
		},
		{
			name: "target job exists, nothing to do",
			args: args{
				jobConfig: &config.JobConfig{
					Periodics: []config.Periodic{
						{
							JobBase: config.JobBase{
								Labels: map[string]string{},
								Name:   prowjobconfigs.CreatePeriodicJobName(semver("1", "21", "0"), "sig-network"),
								Spec: &corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Env: []corev1.EnvVar{
												{
													Name:  "TARGET",
													Value: "k8s-1.21-sig-network",
												},
											},
										},
									},
								},
							},
							Interval: "",
							Cron:     "0 1,9,17 * * *",
							Tags:     nil,
						},
						{
							JobBase: config.JobBase{
								Annotations: map[string]string{},
								Labels:      map[string]string{},
								Name:        prowjobconfigs.CreatePeriodicJobName(semver("1", "22", "0"), "sig-network"),
								Spec: &corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Env: []corev1.EnvVar{
												{
													Name:  "TARGET",
													Value: "k8s-1.22-sig-network",
												},
											},
										},
									},
								},
							},
							Interval: "",
							Cron:     "10 2,10,18 * * *",
							Tags:     nil,
						},
					},
				},
				targetProviderReleaseSemver: semver("1", "22", "0"),
				sourceProviderReleaseSemver: semver("1", "21", "0"),
			},
			wantUpdated: false,
			wantJobConfig: &config.JobConfig{
				Periodics: []config.Periodic{
					{
						JobBase: config.JobBase{
							Labels: map[string]string{},
							Name:   prowjobconfigs.CreatePeriodicJobName(semver("1", "21", "0"), "sig-network"),
							Spec: &corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Env: []corev1.EnvVar{
											{
												Name:  "TARGET",
												Value: "k8s-1.21-sig-network",
											},
										},
									},
								},
							},
						},
						Interval: "",
						Cron:     "0 1,9,17 * * *",
						Tags:     nil,
					},
					{
						JobBase: config.JobBase{
							Annotations: map[string]string{},
							Labels:      map[string]string{},
							Name:        prowjobconfigs.CreatePeriodicJobName(semver("1", "22", "0"), "sig-network"),
							Spec: &corev1.PodSpec{
								Containers: []corev1.Container{
									{
										Env: []corev1.EnvVar{
											{
												Name:  "TARGET",
												Value: "k8s-1.22-sig-network",
											},
										},
									},
								},
							},
						},
						Interval: "",
						Cron:     "10 2,10,18 * * *",
						Tags:     nil,
					},
				},
			},
			wantJobStatesToReportInSerialization: false,
		},
	}
	temp, err := os.MkdirTemp("", "jobconfig")
	panicOn(err)
	defer os.RemoveAll(temp)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotUpdated := copyPeriodicJobsForNewProvider(tt.args.jobConfig, tt.args.targetProviderReleaseSemver, tt.args.sourceProviderReleaseSemver); gotUpdated != tt.wantUpdated {
				t.Errorf("copyPeriodicJobsForNewProvider() = %v, want %v", gotUpdated, tt.wantUpdated)
			}
			if tt.wantUpdated && !reflect.DeepEqual(tt.args.jobConfig, tt.wantJobConfig) {
				t.Errorf("copyPeriodicJobsForNewProvider() = %v", deep.Equal(tt.args.jobConfig, tt.wantJobConfig))
			}
			marshalledConfig, err := yaml.Marshal(&tt.args.jobConfig)
			panicOn(err)
			filePath := path.Join(temp, "periodicsConfig.yaml")
			err = os.WriteFile(filePath, marshalledConfig, os.ModePerm)
			panicOn(err)
			file, err := os.ReadFile(filePath)
			panicOn(err)
			configString := string(file)
			gotJobStatesToReportInSerialization := strings.Contains(configString, "job_states_to_report")
			if tt.wantJobStatesToReportInSerialization != gotJobStatesToReportInSerialization {
				t.Errorf("copyPeriodicJobsForNewProvider(): wantJobStatesToReportInSerialization: want %t, got %t", tt.wantJobStatesToReportInSerialization, gotJobStatesToReportInSerialization)
			}
		})
	}
}

func panicOn(err error) {
	if err != nil {
		panic(err)
	}
}

func semver(major, minor, patch string) *querier.SemVer {
	return &querier.SemVer{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
}

func Test_copyPresubmitJobsForNewProvider(t *testing.T) {
	type args struct {
		jobConfig                   *config.JobConfig
		targetProviderReleaseSemver *querier.SemVer
		sourceProviderReleaseSemver *querier.SemVer
	}
	tests := []struct {
		name          string
		args          args
		wantUpdated   bool
		wantJobConfig *config.JobConfig
	}{
		{
			name: "copies SkipBranches",
			args: args{
				jobConfig: &config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						prowjobconfigs.OrgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, false, false, createJobBrancher([]string{"Release-\\d+\\.\\d+"}, nil)),
						},
					},
				},
				targetProviderReleaseSemver: newMinorSemver("1", "42"),
				sourceProviderReleaseSemver: newMinorSemver("1", "37"),
			},
			wantUpdated: true,
			wantJobConfig: &config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					prowjobconfigs.OrgAndRepoForJobConfig: {
						createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, false, false, createJobBrancher([]string{"Release-\\d+\\.\\d+"}, nil)),
						createPresubmitJobForRelease(newMinorSemver("1", "42"), "sig-network", false, true, false, createJobBrancher([]string{"Release-\\d+\\.\\d+"}, nil)),
					},
				},
			},
		},
		{
			name: "copies Branches",
			args: args{
				jobConfig: &config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						prowjobconfigs.OrgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, false, false, createJobBrancher(nil, []string{"Release-\\d+\\.\\d+"})),
						},
					},
				},
				targetProviderReleaseSemver: newMinorSemver("1", "42"),
				sourceProviderReleaseSemver: newMinorSemver("1", "37"),
			},
			wantUpdated: true,
			wantJobConfig: &config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					prowjobconfigs.OrgAndRepoForJobConfig: {
						createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, false, false, createJobBrancher(nil, []string{"Release-\\d+\\.\\d+"})),
						createPresubmitJobForRelease(newMinorSemver("1", "42"), "sig-network", false, true, false, createJobBrancher(nil, []string{"Release-\\d+\\.\\d+"})),
					},
				},
			},
		},
		{
			name: "target job exists already",
			args: args{
				jobConfig: &config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						prowjobconfigs.OrgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, false, false, createJobBrancher([]string{"Release-\\d+\\.\\d+"}, nil)),
							createPresubmitJobForRelease(newMinorSemver("1", "42"), "sig-network", false, true, false, createJobBrancher([]string{"Release-\\d+\\.\\d+"}, nil)),
						},
					},
				},
				targetProviderReleaseSemver: newMinorSemver("1", "42"),
				sourceProviderReleaseSemver: newMinorSemver("1", "37"),
			},
			wantUpdated: false,
			wantJobConfig: &config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					prowjobconfigs.OrgAndRepoForJobConfig: {
						createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, false, false, createJobBrancher([]string{"Release-\\d+\\.\\d+"}, nil)),
						createPresubmitJobForRelease(newMinorSemver("1", "42"), "sig-network", false, true, false, createJobBrancher([]string{"Release-\\d+\\.\\d+"}, nil)),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotUpdated := copyPresubmitJobsForNewProvider(tt.args.jobConfig, tt.args.targetProviderReleaseSemver, tt.args.sourceProviderReleaseSemver); gotUpdated != tt.wantUpdated {
				t.Errorf("copyPresubmitJobsForNewProvider() = %v, want %v", gotUpdated, tt.wantUpdated)
			}
			if !reflect.DeepEqual(tt.args.jobConfig, tt.wantJobConfig) {
				t.Errorf("copyPresubmitJobsForNewProvider(), diff: %v",
					deep.Equal(tt.args.jobConfig, tt.wantJobConfig))
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

func createPresubmitJobForRelease(semver *querier.SemVer, sigName string, alwaysRun, optional, skipReport bool, brancher config.Brancher) config.Presubmit {
	res := config.Presubmit{
		AlwaysRun: alwaysRun,
		Optional:  optional,
		JobBase: config.JobBase{
			Annotations: map[string]string{
				"fork-per-Release":    "true",
				"testgrid-dashboards": "kubevirt-presubmits",
			},
			Labels: map[string]string{
				"preset-docker-mirror-proxy": "true",
				"preset-shared-images":       "true",
			},
			Name: prowjobconfigs.CreatePresubmitJobName(semver, sigName),
			Spec: &corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Env: []corev1.EnvVar{
							{
								Name:  "WHATEVER",
								Value: "should stay unchanged",
							},
							{
								Name:  "TARGET",
								Value: prowjobconfigs.CreateTargetValue(semver, sigName),
							},
						},
					},
				},
			},
		},
		Reporter: config.Reporter{
			SkipReport: skipReport,
		},
		Brancher: brancher,
	}
	return res
}

func createJobBrancher(skipBranches []string, branches []string) config.Brancher {
	return config.Brancher{
		SkipBranches: skipBranches,
		Branches:     branches,
	}
}

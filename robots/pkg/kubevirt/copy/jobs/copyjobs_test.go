package jobs

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"sigs.k8s.io/yaml"
	"strings"
	"testing"

	"github.com/go-test/deep"
	"github.com/google/go-github/github"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/config"

	"kubevirt.io/project-infra/robots/pkg/querier"
)

func Test_advanceCronExpression(t *testing.T) {
	type args struct {
		sourceCronExpr string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "zero one nine seventeen",
			args: args{
				sourceCronExpr: "0 1,9,17 * * *",
			},
			want: "10 2,10,18 * * *",
		},
		{
			name: "fifty one nine seventeen",
			args: args{
				sourceCronExpr: "50 1,9,17 * * *",
			},
			want: "0 2,10,18 * * *",
		},
		{
			name: "zero eight sixteen twentyfour",
			args: args{
				sourceCronExpr: "0 8,16,24 * * *",
			},
			want: "10 1,9,17 * * *",
		},
		{
			name: "zero seven fifteen twentythree",
			args: args{
				sourceCronExpr: "0 7,15,23 * * *",
			},
			want: "10 0,8,16 * * *",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := advanceCronExpression(tt.args.sourceCronExpr); got != tt.want {
				t.Errorf("advanceCronExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
			name: "has one patch release for latest",
			args: args{
				releases: []*github.RepositoryRelease{
					release("v1.22.0"),
					release("v1.21.3"),
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
					release("v1.22.1"),
					release("v1.22.0"),
					release("v1.21.3"),
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
			name: "has one release only, should err",
			args: args{
				releases: []*github.RepositoryRelease{
					release("v1.22.1"),
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
					release("v1.22.1"),
					release("v1.22.0"),
				},
			},
			wantTargetRelease: &querier.SemVer{
				Major: "1",
				Minor: "22",
				Patch: "1",
			},
			wantSourceRelease: nil,
			wantErr:           fmt.Errorf("no source release found"),
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

func TestCopyPeriodicJobsForNewProvider(t *testing.T) {
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
								Name: createPeriodicJobName(semver("1", "21", "0"), "sig-network"),
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
							Name:   createPeriodicJobName(semver("1", "21", "0"), "sig-network"),
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
							Name:        createPeriodicJobName(semver("1", "22", "0"), "sig-network"),
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
								Name: createPeriodicJobName(semver("1", "21", "0"), "sig-network"),
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
											Org:            "kubevirt",
											Repo:           "kubevirt",
											BaseRef:        "main",
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
							Name:   createPeriodicJobName(semver("1", "21", "0"), "sig-network"),
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
										Org:            "kubevirt",
										Repo:           "kubevirt",
										BaseRef:        "main",
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
							Name:        createPeriodicJobName(semver("1", "22", "0"), "sig-network"),
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
										Org:            "kubevirt",
										Repo:           "kubevirt",
										BaseRef:        "main",
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
			if gotUpdated := CopyPeriodicJobsForNewProvider(tt.args.jobConfig, tt.args.targetProviderReleaseSemver, tt.args.sourceProviderReleaseSemver); gotUpdated != tt.wantUpdated {
				t.Errorf("CopyPeriodicJobsForNewProvider() = %v, want %v", gotUpdated, tt.wantUpdated)
			}
			if tt.wantUpdated && !reflect.DeepEqual(tt.args.jobConfig, tt.wantJobConfig) {
				t.Errorf("CopyPeriodicJobsForNewProvider() = %v", deep.Equal(tt.args.jobConfig, tt.wantJobConfig))
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
				t.Errorf("CopyPeriodicJobsForNewProvider(): wantJobStatesToReportInSerialization: want %t, got %t", tt.wantJobStatesToReportInSerialization, gotJobStatesToReportInSerialization)
			}
		})
	}
}

func panicOn(err error) {
	if err != nil {
		panic(err)
	}
}

func release(version string) *github.RepositoryRelease {
	result := github.RepositoryRelease{}
	result.TagName = &version
	return &result
}

func semver(major, minor, patch string) *querier.SemVer {
	return &querier.SemVer{
		Major: major,
		Minor: minor,
		Patch: patch,
	}
}

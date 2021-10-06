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
	"crypto/sha256"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/go-test/deep"
	"github.com/google/go-github/github"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/test-infra/prow/apis/prowjobs/v1"
	"k8s.io/test-infra/prow/config"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/prowjobconfigs"
	"kubevirt.io/project-infra/robots/pkg/querier"
)

func Test_ensureLatestJobsAreRequired(t *testing.T) {
	type args struct {
		jobConfigKubevirtPresubmits config.JobConfig
		release                     *querier.SemVer
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
						prowjobconfigs.OrgAndRepoForJobConfig: {
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
						prowjobconfigs.OrgAndRepoForJobConfig: {
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
			want: ALL_JOBS_ARE_REQUIRED,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, message := ensureSigJobsAreRequired(tt.args.jobConfigKubevirtPresubmits, tt.args.release)
			if got != tt.want {
				t.Errorf("ensureSigJobsAreRequired() = %v, want %v", got, tt.want)
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
			name: "jobs missing",
			args: args{
				jobConfigKubevirtPresubmits: config.JobConfig{},
				requiredReleases: []*querier.SemVer{
					newMinorSemver("1", "37"),
					newMinorSemver("1", "42"),
				},
			},
			wantAllJobsExist: false,
		},
		{
			name: "some jobs are missing",
			args: args{
				jobConfigKubevirtPresubmits: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						prowjobconfigs.OrgAndRepoForJobConfig: {
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-network", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "sig-compute", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "37"), "operator", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "42"), "sig-network", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "42"), "sig-compute", true, false, false),
							createPresubmitJobForRelease(newMinorSemver("1", "42"), "sig-storage", true, false, false),
						},
					},
				},
				requiredReleases: []*querier.SemVer{
					newMinorSemver("1", "37"),
					newMinorSemver("1", "42"),
				},
			},
			wantAllJobsExist: false,
		},
		{
			name: "all jobs exist",
			args: args{
				jobConfigKubevirtPresubmits: config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						prowjobconfigs.OrgAndRepoForJobConfig: {
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
			gotAllJobsExist, gotMessage := ensureSigPresubmitJobsExistForReleases(tt.args.jobConfigKubevirtPresubmits, tt.args.requiredReleases)
			if gotAllJobsExist != tt.wantAllJobsExist {
				t.Errorf("ensureSigPresubmitJobsExistForReleases() gotAllJobsExist = %v, want %v", gotAllJobsExist, tt.wantAllJobsExist)
			}
			t.Logf("message: %s", gotMessage)
		})
	}
}

func Test_deletePeriodicJobsForRelease(t *testing.T) {
	type args struct {
		jobConfig *config.JobConfig
		release   *querier.SemVer
	}
	tests := []struct {
		name          string
		args          args
		wantUpdated   bool
		wantJobConfig *config.JobConfig
	}{
		{
			name: "no jobs to delete",
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
								Name: prowjobconfigs.CreatePeriodicJobName(newMinorSemver("1", "20"), "sig-network"),
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
				release: newMinorSemver("1", "19"),
			},
			wantUpdated: false,
		},
		{
			name: "one job to delete",
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
								Name: prowjobconfigs.CreatePeriodicJobName(newMinorSemver("1", "20"), "sig-network"),
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
								Labels: map[string]string{},
								ReporterConfig: &v1.ReporterConfig{
									Slack: &v1.SlackReporterConfig{
										JobStatesToReport: []v1.ProwJobState{},
									},
								},
								Name: prowjobconfigs.CreatePeriodicJobName(newMinorSemver("1", "19"), "sig-network"),
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
				release: newMinorSemver("1", "19"),
			},
			wantUpdated: true,
			wantJobConfig: &config.JobConfig{
				Periodics: []config.Periodic{
					{
						JobBase: config.JobBase{
							Labels: map[string]string{},
							ReporterConfig: &v1.ReporterConfig{
								Slack: &v1.SlackReporterConfig{
									JobStatesToReport: []v1.ProwJobState{},
								},
							},
							Name: prowjobconfigs.CreatePeriodicJobName(newMinorSemver("1", "20"), "sig-network"),
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deleteSigPeriodicJobsForRelease(tt.args.jobConfig, tt.args.release); got != tt.wantUpdated {
				t.Errorf("deleteSigPeriodicJobsForRelease() = %v, want %v", got, tt.wantUpdated)
			}
			if tt.wantUpdated && !reflect.DeepEqual(tt.args.jobConfig, tt.wantJobConfig) {
				t.Errorf("deleteSigPeriodicJobsForRelease() = %v", deep.Equal(tt.args.jobConfig, tt.wantJobConfig))
			}
		})
	}
}

func Test_deletePresubmitJobsForRelease(t *testing.T) {
	type args struct {
		jobConfig     *config.JobConfig
		targetRelease *querier.SemVer
	}
	tests := []struct {
		name          string
		args          args
		wantUpdated   bool
		wantJobConfig *config.JobConfig
	}{
		{
			name: "no jobs to delete",
			args: args{
				jobConfig: &config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						prowjobconfigs.OrgAndRepoForJobConfig: {
							{
								JobBase: config.JobBase{
									Labels: map[string]string{},
									ReporterConfig: &v1.ReporterConfig{
										Slack: &v1.SlackReporterConfig{
											JobStatesToReport: []v1.ProwJobState{},
										},
									},
									Name: prowjobconfigs.CreatePresubmitJobName(newMinorSemver("1", "20"), "sig-network"),
									Spec: &corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Env: []corev1.EnvVar{},
											},
										},
									},
								},
							},
						},
					},
				},
				targetRelease: newMinorSemver("1", "19"),
			},
			wantUpdated: false,
		},
		{
			name: "one job to delete",
			args: args{
				jobConfig: &config.JobConfig{
					PresubmitsStatic: map[string][]config.Presubmit{
						prowjobconfigs.OrgAndRepoForJobConfig: {
							{
								JobBase: config.JobBase{
									Labels: map[string]string{},
									ReporterConfig: &v1.ReporterConfig{
										Slack: &v1.SlackReporterConfig{
											JobStatesToReport: []v1.ProwJobState{},
										},
									},
									Name: prowjobconfigs.CreatePresubmitJobName(newMinorSemver("1", "20"), "sig-network"),
									Spec: &corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Env: []corev1.EnvVar{},
											},
										},
									},
								},
							},
							{
								JobBase: config.JobBase{
									Labels: map[string]string{},
									ReporterConfig: &v1.ReporterConfig{
										Slack: &v1.SlackReporterConfig{
											JobStatesToReport: []v1.ProwJobState{},
										},
									},
									Name: prowjobconfigs.CreatePresubmitJobName(newMinorSemver("1", "19"), "sig-network"),
									Spec: &corev1.PodSpec{
										Containers: []corev1.Container{
											{
												Env: []corev1.EnvVar{},
											},
										},
									},
								},
							},
						},
					},
				},
				targetRelease: newMinorSemver("1", "19"),
			},
			wantUpdated: true,
			wantJobConfig: &config.JobConfig{
				PresubmitsStatic: map[string][]config.Presubmit{
					prowjobconfigs.OrgAndRepoForJobConfig: {
						{
							JobBase: config.JobBase{
								Labels: map[string]string{},
								ReporterConfig: &v1.ReporterConfig{
									Slack: &v1.SlackReporterConfig{
										JobStatesToReport: []v1.ProwJobState{},
									},
								},
								Name: prowjobconfigs.CreatePresubmitJobName(newMinorSemver("1", "20"), "sig-network"),
								Spec: &corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Env: []corev1.EnvVar{},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deleteSigPresubmitJobsForRelease(tt.args.jobConfig, tt.args.targetRelease); got != tt.wantUpdated {
				t.Errorf("deleteSigPresubmitJobsForRelease() = %v, wantUpdated %v", got, tt.wantUpdated)
			}
			if tt.wantUpdated && !reflect.DeepEqual(tt.args.jobConfig, tt.wantJobConfig) {
				t.Errorf("deleteSigPresubmitJobsForRelease() = %v", deep.Equal(tt.args.jobConfig, tt.wantJobConfig))
			}
		})
	}
}

func Test_removeOldJobsIfNewOnesExist(t *testing.T) {
	type args struct {
		releases       []*github.RepositoryRelease
		removeJobsOpts removeJobsOptions
	}
	tests := []struct {
		name             string
		args             args
		wantModification bool
	}{
		{
			name: "should modify",
			args: args{
				releases: []*github.RepositoryRelease{
					newRelease("v1.22.0"),
					newRelease("v1.21.0"),
					newRelease("v1.20.0"),
					newRelease("v1.19.0"),
				},
				removeJobsOpts: removeJobsOptions{
					jobConfigPathKubevirtPeriodics:  "testdata/should_modify/kubevirt-periodics.yaml",
					jobConfigPathKubevirtPresubmits: "testdata/should_modify/kubevirt-presubmits.yaml",
				},
			},
			wantModification: true,
		},
		{
			name: "should not modify",
			args: args{
				releases: []*github.RepositoryRelease{
					newRelease("v1.22.0"),
					newRelease("v1.21.0"),
					newRelease("v1.20.0"),
					newRelease("v1.19.0"),
				},
				removeJobsOpts: removeJobsOptions{
					jobConfigPathKubevirtPeriodics:  "testdata/should_not_modify/kubevirt-periodics.yaml",
					jobConfigPathKubevirtPresubmits: "testdata/should_not_modify/kubevirt-presubmits.yaml",
				},
			},
			wantModification: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir, err := os.MkdirTemp("", "removeOldJobsIfNewOnesExist-")
			checkErr(err)
			defer os.RemoveAll(tempDir)

			copyFiles([]string{tt.args.removeJobsOpts.jobConfigPathKubevirtPeriodics, tt.args.removeJobsOpts.jobConfigPathKubevirtPresubmits}, tempDir)
			removeJobsOpts = removeJobsOptions{
				jobConfigPathKubevirtPeriodics:  filepath.Join(tempDir, filepath.Base(tt.args.removeJobsOpts.jobConfigPathKubevirtPeriodics)),
				jobConfigPathKubevirtPresubmits: filepath.Join(tempDir, filepath.Base(tt.args.removeJobsOpts.jobConfigPathKubevirtPresubmits)),
			}
			err = removeOldJobsIfNewOnesExist(tt.args.releases)
			if err != nil {
				t.Errorf("removeOldJobsIfNewOnesExist(), unexpected error %v", err)
			}

			sum256OrigPeriodics := hashFile(tt.args.removeJobsOpts.jobConfigPathKubevirtPeriodics)
			sum256TempCopyPeriodics := hashFile(removeJobsOpts.jobConfigPathKubevirtPeriodics)
			sum256OrigPresubmits := hashFile(tt.args.removeJobsOpts.jobConfigPathKubevirtPresubmits)
			sum256TempCopyPresubmits := hashFile(removeJobsOpts.jobConfigPathKubevirtPresubmits)

			if tt.wantModification {
				if reflect.DeepEqual(sum256OrigPeriodics, sum256TempCopyPeriodics) {
					t.Errorf("removeOldJobsIfNewOnesExist(), wantModification %v, got %v", tt.wantModification, false)
				}
				if reflect.DeepEqual(sum256OrigPresubmits, sum256TempCopyPresubmits) {
					t.Errorf("removeOldJobsIfNewOnesExist(), wantModification %v, got %v", tt.wantModification, false)
				}
			} else {
				if !reflect.DeepEqual(sum256OrigPeriodics, sum256TempCopyPeriodics) {
					t.Errorf("removeOldJobsIfNewOnesExist(), wantModification %v, got %v", tt.wantModification, true)
				}
				if !reflect.DeepEqual(sum256OrigPresubmits, sum256TempCopyPresubmits) {
					t.Errorf("removeOldJobsIfNewOnesExist(), wantModification %v, got %v", tt.wantModification, true)
				}
			}
		})
	}
}

func hashFile(filePath string) [32]byte {
	file, err := os.ReadFile(filePath)
	checkErr(err)
	return sha256.Sum256(file)
}

func copyFiles(srcPaths []string, destDir string) {
	for _, srcPath := range srcPaths {
		open, err := os.Open(srcPath)
		checkErr(err)
		baseName := filepath.Base(srcPath)
		create, err := os.Create(filepath.Join(destDir, baseName))
		checkErr(err)
		_, err = io.Copy(create, open)
		checkErr(err)
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
	return config.Presubmit{
		AlwaysRun: alwaysRun,
		Optional:  optional,
		JobBase: config.JobBase{
			Name: prowjobconfigs.CreatePresubmitJobName(semver, sigName),
		},
		Reporter: config.Reporter{
			SkipReport: skipReport,
		},
	}
}

func newRelease(version string) *github.RepositoryRelease {
	result := github.RepositoryRelease{}
	result.TagName = &version
	return &result
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

package cmd

import (
	"github.com/bndr/gojenkins"
	"github.com/joshdk/go-junit"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"kubevirt.io/project-infra/robots/pkg/flakefinder"
	"kubevirt.io/project-infra/robots/pkg/flakefinder/build"
	"kubevirt.io/project-infra/robots/pkg/validation"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func Test_fetchJunitFilesFromArtifacts(t *testing.T) {
	type args struct {
		completedBuilds []*gojenkins.Build
		fLog            *log.Entry
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "fetches all relevant artifacts",
			args: args{
				completedBuilds: []*gojenkins.Build{
					{
						Raw: &gojenkins.BuildResponse{
							Artifacts: []struct {
								DisplayPath  string `json:"displayPath"`
								FileName     string `json:"fileName"`
								RelativePath string `json:"relativePath"`
							}{
								{
									FileName: "footest.xml",
								},
								{
									FileName: "junit.functest.xml",
								},
								{
									FileName: "partial.junit.functest.1.xml",
								},
								{
									FileName: "partial.junit.functest.2.xml",
								},
								{
									FileName: "partial.junit.functest.3.xml",
								},
								{
									FileName: "bartest.xml",
								},
								{
									FileName: "merged.junit.functest.xml",
								},
								{
									FileName: "foobar.junit.functest.xml",
								},
							},
						},
						Job:     nil,
						Jenkins: nil,
						Base:    "",
						Depth:   0,
					},
				},
				fLog: log.StandardLogger().WithField("test", "test"),
			},
			want: []string{
				"junit.functest.xml",
				"partial.junit.functest.1.xml",
				"partial.junit.functest.2.xml",
				"partial.junit.functest.3.xml",
				"merged.junit.functest.xml",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fetchJunitFilesFromArtifacts(tt.args.completedBuilds, tt.args.fLog)
			actualFileNames := []string{}
			for _, artifact := range got {
				actualFileNames = append(actualFileNames, artifact.FileName)
			}
			if !reflect.DeepEqual(tt.want, actualFileNames) {
				t.Errorf("fetchJunitFilesFromArtifacts() = %v, want %v", actualFileNames, tt.want)
			}
		})
	}
}

func Test_writeReportToFileProducesValidOutput(t *testing.T) {
	type args struct {
		startOfReport time.Time
		endOfReport   time.Time
		reports       []*flakefinder.JobResult
		validators    []validation.ContentValidator
		ratings       []build.Rating
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "one job - failed tests",
			args: args{
				startOfReport: time.Now(),
				endOfReport:   time.Now(),
				reports: []*flakefinder.JobResult{
					{
						Job: "job1",
						JUnit: []junit.Suite{
							{
								Name:       "Suite 1",
								Package:    "test.blah.package",
								Properties: nil,
								Tests: []junit.Test{
									{
										Name:       "Test 1",
										Classname:  "testClass",
										Duration:   37,
										Status:     junit.StatusFailed,
										Error:      nil,
										Properties: nil,
									},
									{
										Name:       "Test 2",
										Classname:  "testClass",
										Duration:   37,
										Status:     junit.StatusError,
										Error:      nil,
										Properties: nil,
									},
								},
								SystemOut: "",
								SystemErr: "",
								Totals: junit.Totals{
									Tests:    42,
									Passed:   37,
									Skipped:  17,
									Failed:   11,
									Error:    3,
									Duration: 1742,
								},
							},
						},
						BuildNumber: 1,
						PR:          42,
					},
				},
				validators: []validation.ContentValidator{
					validation.HTMLValidator{},
					validation.JSONValidator{},
				},
			},
		},
		{
			name: "two jobs - failed tests",
			args: args{
				startOfReport: time.Now(),
				endOfReport:   time.Now(),
				reports: []*flakefinder.JobResult{
					{
						Job: "job1",
						JUnit: []junit.Suite{
							{
								Name:       "Suite 1",
								Package:    "test.blah.package",
								Properties: nil,
								Tests: []junit.Test{
									{
										Name:       "Test 1",
										Classname:  "testClass",
										Duration:   37,
										Status:     junit.StatusFailed,
										Error:      nil,
										Properties: nil,
									},
									{
										Name:       "Test 2",
										Classname:  "testClass",
										Duration:   37,
										Status:     junit.StatusError,
										Error:      nil,
										Properties: nil,
									},
								},
								SystemOut: "",
								SystemErr: "",
								Totals: junit.Totals{
									Tests:    42,
									Passed:   37,
									Skipped:  17,
									Failed:   11,
									Error:    3,
									Duration: 1742,
								},
							},
						},
						BuildNumber: 1,
						PR:          42,
					},
					{
						Job: "job2",
						JUnit: []junit.Suite{
							{
								Name:       "Suite 1",
								Package:    "test.blah.package",
								Properties: nil,
								Tests: []junit.Test{
									{
										Name:       "Test 3",
										Classname:  "testClass",
										Duration:   37,
										Status:     junit.StatusFailed,
										Error:      nil,
										Properties: nil,
									},
									{
										Name:       "Test 4",
										Classname:  "testClass",
										Duration:   37,
										Status:     junit.StatusError,
										Error:      nil,
										Properties: nil,
									},
								},
								SystemOut: "",
								SystemErr: "",
								Totals: junit.Totals{
									Tests:    42,
									Passed:   37,
									Skipped:  17,
									Failed:   11,
									Error:    3,
									Duration: 1742,
								},
							},
						},
						BuildNumber: 1,
						PR:          42,
					},
				},
				validators: []validation.ContentValidator{
					validation.HTMLValidator{},
					validation.JSONValidator{},
				},
			},
		},
	}

	tempDir, err := ioutil.TempDir("", "reportFile")
	if err != nil {
		t.Errorf("failed to create temp report file: %v", err)
		return
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile := filepath.Join(tempDir, "report.html")
			writeReportToFile(tt.args.startOfReport, tt.args.endOfReport, tt.args.reports, tempFile, tt.args.ratings)

			for _, currentValidator := range tt.args.validators {
				targetFileName := currentValidator.GetTargetFileName(tempFile)
				content, err := ioutil.ReadFile(targetFileName)
				if err != nil {
					t.Errorf("failed to read temp report file: %v", err)
					return
				}

				err = currentValidator.IsValid(content)
				if err != nil {
					t.Errorf("Report:\n%s\n\nfailed to validate report file: %v", targetFileName, err)
				}
			}
		})
	}
}

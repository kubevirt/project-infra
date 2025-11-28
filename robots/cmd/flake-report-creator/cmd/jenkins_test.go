package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"kubevirt.io/project-infra/pkg/flakefinder"
	"kubevirt.io/project-infra/pkg/flakefinder/build"
	"kubevirt.io/project-infra/pkg/validation"

	"github.com/bndr/gojenkins"
	"github.com/joshdk/go-junit"
	log "github.com/sirupsen/logrus"
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
		ratings       []build.Rating
		reports       []*flakefinder.JobResult
		validators    []validation.ContentValidator
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
				ratings: []build.Rating{
					{
						Name:         "test-build-1",
						Source:       "https://source/test-build-1",
						StartFrom:    24 * time.Hour,
						BuildNumbers: []int64{int64(17), int64(42)},
						BuildNumbersToData: map[int64]build.BuildData{
							int64(17): {
								Number:   int64(17),
								Failures: int64(99),
								Sigma:    float64(4.0),
							},
							int64(42): {
								Number:   int64(42),
								Failures: int64(13),
								Sigma:    float64(2.0),
							},
						},
						TotalCompletedBuilds: 2,
						TotalFailures:        131,
						Mean:                 17,
						Variance:             23,
						StandardDeviation:    1.23,
					},
				},
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
				ratings: []build.Rating{
					{
						Name:         "test-build-1",
						Source:       "https://source/test-build-1",
						StartFrom:    24 * time.Hour,
						BuildNumbers: []int64{int64(17), int64(42)},
						BuildNumbersToData: map[int64]build.BuildData{
							int64(17): {
								Number:   int64(17),
								Failures: int64(99),
								Sigma:    float64(4.0),
							},
							int64(42): {
								Number:   int64(42),
								Failures: int64(13),
								Sigma:    float64(2.0),
							},
						},
						TotalCompletedBuilds: 2,
						TotalFailures:        131,
						Mean:                 17,
						Variance:             23,
						StandardDeviation:    1.23,
					},
				},
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

	tempDir, err := os.MkdirTemp("", "reportFile")
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
				content, err := os.ReadFile(targetFileName)
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

func Test_writeReportToFileCreatesTags(t *testing.T) {
	type args struct {
		startOfReport time.Time
		endOfReport   time.Time
		reports       []*flakefinder.JobResult
		ratings       []build.Rating
		expectations  []expectation
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "one job",
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
										Name:       "[Serial][sig-operator]virt-handler canary upgrade [QUARANTINE]should successfully upgrade virt-handler",
										Classname:  "testClass",
										Duration:   37,
										Status:     junit.StatusFailed,
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
				expectations: []expectation{
					expectContains("Serial"),
					expectContains("sig-operator"),
					expectContains("QUARANTINE"),
				},
			},
		},
	}

	tempDir, err := os.MkdirTemp("", "reportFile")
	if err != nil {
		t.Errorf("failed to create temp report file: %v", err)
		return
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempFile := filepath.Join(tempDir, "report.html")
			writeReportToFile(tt.args.startOfReport, tt.args.endOfReport, tt.args.reports, tempFile, tt.args.ratings)

			content, err := os.ReadFile(tempFile)
			if err != nil {
				t.Errorf("failed to read temp report file: %v", err)
				return
			}

			for _, expectation := range tt.args.expectations {
				expectation.contains(string(content), t)
			}
		})
	}
}

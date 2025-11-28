package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/joshdk/go-junit"
	"kubevirt.io/project-infra/pkg/flakefinder"
	"kubevirt.io/project-infra/pkg/validation"
)

func Test_writeProwReportToFileProducesValidOutput(t *testing.T) {
	type args struct {
		startOfReport time.Time
		endOfReport   time.Time
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
			baseReportFile := filepath.Join(tempDir, "report.html")
			tempFile, err := os.OpenFile(baseReportFile, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				t.Errorf("failed to create temp report file: %v", err)
				return
			}
			err = writeProwReportToFile(tt.args.startOfReport, tt.args.reports, tempFile)
			if err != nil {
				t.Errorf("failed to write report to file: %v", err)
				return
			}

			for _, currentValidator := range tt.args.validators {
				targetFileName := currentValidator.GetTargetFileName(baseReportFile)
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

func Test_writeProwReportToFileCreatesTags(t *testing.T) {
	type args struct {
		startOfReport time.Time
		endOfReport   time.Time
		reports       []*flakefinder.JobResult
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
			baseReportFile := filepath.Join(tempDir, "report.html")
			tempFile, err := os.OpenFile(baseReportFile, os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				t.Errorf("failed to create temp report file: %v", err)
				return
			}
			writeProwReportToFile(tt.args.startOfReport, tt.args.reports, tempFile)

			content, err := os.ReadFile(baseReportFile)
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

package cmd

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/bndr/gojenkins"
	"github.com/joshdk/go-junit"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"kubevirt.io/project-infra/robots/pkg/flakefinder"
	"kubevirt.io/project-infra/robots/pkg/flakefinder/build"
	"path/filepath"
	"reflect"
	"strings"
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

type contentValidator interface {
	isValid(content []byte) error
	getTargetFileName(filename string) string
}

type jSONValidator struct{}

func (j jSONValidator) isValid(content []byte) error {
	if json.Valid(content) {
		return nil
	}
	return fmt.Errorf("json invalid:\n%s", string(content))
}

func (j jSONValidator) getTargetFileName(filename string) string {
	return strings.TrimSuffix(filename, ".html") + ".json"
}

type hTMLValidator struct{}

func (j hTMLValidator) isValid(content []byte) error {
	stringContent := string(content)
	r := strings.NewReader(stringContent)
	d := xml.NewDecoder(r)

	d.Strict = true
	d.Entity = xml.HTMLEntity
	for {
		_, err := d.Token()
		switch err {
		case io.EOF:
			return nil
		case nil:
		default:
			return fmt.Errorf("Report:\n%s\n\nfailed to validate report file: %v", stringContent, err)
		}
	}
}

func (j hTMLValidator) getTargetFileName(filename string) string {
	return filename
}

func Test_writeReportToFileProducesValidOutput(t *testing.T) {
	type args struct {
		startOfReport time.Time
		endOfReport   time.Time
		reports       []*flakefinder.JobResult
		validators    []contentValidator
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
				validators: []contentValidator{
					hTMLValidator{},
					jSONValidator{},
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
				validators: []contentValidator{
					hTMLValidator{},
					jSONValidator{},
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
				targetFileName := currentValidator.getTargetFileName(tempFile)
				content, err := ioutil.ReadFile(targetFileName)
				if err != nil {
					t.Errorf("failed to read temp report file: %v", err)
					return
				}

				err = currentValidator.isValid(content)
				if err != nil {
					t.Errorf("Report:\n%s\n\nfailed to validate report file: %v", targetFileName, err)
				}
			}
		})
	}
}

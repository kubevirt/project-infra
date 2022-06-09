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
	"os"
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

func Test_writeReportToFile(t *testing.T) {
	type args struct {
		startOfReport time.Time
		endOfReport   time.Time
		reports       []*flakefinder.JobResult
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
			},
		},
	}

	tempFile, err := ioutil.TempFile("", "reportFile")
	if err != nil {
		t.Errorf("failed to create temp report file: %v", err)
		return
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeReportToFile(tt.args.startOfReport, tt.args.endOfReport, tt.args.reports, tempFile.Name())

			// validate output is valid xml
			file, err := ioutil.ReadFile(tempFile.Name())
			if err != nil {
				t.Errorf("failed to read temp report file: %v", err)
				return
			}
			r := strings.NewReader(string(file))
			d := xml.NewDecoder(r)

			d.Strict = true
			d.Entity = xml.HTMLEntity
			for {
				_, err := d.Token()
				switch err {
				case io.EOF:
					return
				case nil:
				default:
					t.Errorf("Report:\n%s\n\nfailed to validate report file: %v", string(file), err)
					return
				}
			}
		})
	}
}

func Test_writeReportToOutputFile(t *testing.T) {
	type args struct {
		outputFile     string
		reportTemplate string
		params         flakefinder.Params
		validator      func([]byte) error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test report creation",
			args: args{
				outputFile: "test.html",
				params: flakefinder.Params{
					Data: map[string]map[string]*flakefinder.Details{
						"test1": {
							"blah1": &flakefinder.Details{
								Succeeded: 1,
								Skipped:   2,
								Failed:    3,
								Severity:  "SEVERE",
								Jobs: []*flakefinder.Job{
									{
										BuildNumber: 1742,
										Severity:    "SEVERE",
										PR:          4217,
										Job:         "asdhfkfsaj",
									},
									{
										BuildNumber: 1742,
										Severity:    "SEVERE",
										PR:          4217,
										Job:         "asdhfkfsaj",
									},
								},
							},
							"blah2": &flakefinder.Details{
								Succeeded: 1,
								Skipped:   2,
								Failed:    3,
								Severity:  "SEVERE",
								Jobs: []*flakefinder.Job{
									{
										BuildNumber: 1742,
										Severity:    "SEVERE",
										PR:          4217,
										Job:         "asdhfkfsaj",
									},
								},
							},
						},
					},
				},
				validator: func(content []byte) error {
					if json.Valid(content) {
						return nil
					}
					return fmt.Errorf("json invalid:\n%s", string(content))
				},
			},
		},
	}
	dir, err := ioutil.TempDir("", "Test_writeReportToOutputFile")
	if err != nil {
		t.Errorf("failed to create tempdir: %v", err)
	}
	defer os.RemoveAll(dir)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outputFile := filepath.Join(dir, tt.args.outputFile)
			writeReportsToOutputFiles(outputFile, tt.args.params)
			_, err2 := os.Stat(outputFile)
			if err2 != nil {
				t.Errorf("failed to access outputFile %s: %v", outputFile, err2)
			}
			if tt.args.validator != nil {
				bytes, err2 := os.ReadFile(strings.TrimSuffix(outputFile, ".html") + ".json")
				t.Logf("output file %q:\n%s", outputFile, string(bytes))
				if err2 != nil {
					t.Errorf("failed to read output file %q: %v", outputFile, err2)
				}
				err2 = tt.args.validator(bytes)
				if err2 != nil {
					t.Errorf("failed to validate output file %q: %v", outputFile, err2)
				}
			}
		})
	}
}

package main

import (
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"reflect"
	"regexp"
	"testing"
)

func Test_writeHTMLReportToOutput(t *testing.T) {
	type args struct {
		htmlReportOutputWriter       io.Writer
		testNames                    []string
		filteredTestNames            []string
		skippedTests                 map[string]interface{}
		lookedAtJobs                 []string
		testNamesToJobNamesToSkipped map[string]map[string]int
		err                          error
		jLog                         *logrus.Entry
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "test template",
			args: args{
				htmlReportOutputWriter: os.Stdout,
				testNames:              []string{"a", "b", "c"},
				filteredTestNames:      []string{"la", "le", "lu"},
				skippedTests: map[string]interface{}{
					"a": struct{}{}},
				lookedAtJobs: []string{"job1", "job2", "job3"},
				testNamesToJobNamesToSkipped: map[string]map[string]int{
					"a": {
						"job1": test_execution_skipped,
						"job2": test_execution_skipped,
					},
					"b": {
						"job1": test_execution_skipped,
						"job2": test_execution_run,
						"job3": test_execution_run,
					},
					"c": {
						"job1": test_execution_skipped,
						"job2": test_execution_skipped,
						"job3": test_execution_run,
					},
				},
				err:  nil,
				jLog: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writeHTMLReportToOutput(newData(tt.args.testNames, tt.args.filteredTestNames, tt.args.skippedTests, tt.args.lookedAtJobs, tt.args.testNamesToJobNamesToSkipped), tt.args.htmlReportOutputWriter)
		})
	}
}

func Test_createReportData(t *testing.T) {
	type args struct {
		testNameFilterRegexp                 *regexp.Regexp
		testNamesToJobNamesToExecutionStatus map[string]map[string]int
	}
	tests := []struct {
		name string
		args args
		want Data
	}{
		{
			name: "test has no data",
			args: args{
				testNameFilterRegexp: regexp.MustCompile("blah"),
				testNamesToJobNamesToExecutionStatus: map[string]map[string]int{
					"testName": {
						"jobName": test_execution_no_data,
					},
				},
			},
			want: newData(
				[]string{"testName"}, // testNames
				[]string{},           // filteredTestNames
				map[string]interface{}{"testName": struct{}{}}, // skippedTests
				[]string{"jobName"},                            // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"jobName": test_execution_no_data,
					},
				}, // testNamesToJobNamesToExecutionStatus
			),
		},
		{
			name: "test is run",
			args: args{
				testNameFilterRegexp: regexp.MustCompile("blah"),
				testNamesToJobNamesToExecutionStatus: map[string]map[string]int{
					"testName": {
						"jobName": test_execution_run,
					},
				},
			},
			want: newData(
				[]string{"testName"},     // testNames
				[]string{},               // filteredTestNames
				map[string]interface{}{}, // skippedTests
				[]string{"jobName"},      // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"jobName": test_execution_run,
					},
				}, // testNamesToJobNamesToExecutionStatus
			),
		},
		{
			name: "test is skipped",
			args: args{
				testNameFilterRegexp: regexp.MustCompile("blah"),
				testNamesToJobNamesToExecutionStatus: map[string]map[string]int{
					"testName": {
						"jobName": test_execution_skipped,
					},
				},
			},
			want: newData(
				[]string{"testName"}, // testNames
				[]string{},           // filteredTestNames
				map[string]interface{}{"testName": struct{}{}}, // skippedTests
				[]string{"jobName"},                            // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"jobName": test_execution_skipped,
					},
				}, // testNamesToJobNamesToExecutionStatus
			),
		},
		{
			name: "test is filtered",
			args: args{
				testNameFilterRegexp: regexp.MustCompile("testName"),
				testNamesToJobNamesToExecutionStatus: map[string]map[string]int{
					"testName": {
						"jobName": test_execution_skipped,
					},
				},
			},
			want: newData(
				[]string{},               // testNames
				[]string{"testName"},     // filteredTestNames
				map[string]interface{}{}, // skippedTests
				[]string{},               // lookedAtJobs
				map[string]map[string]int{
					"testName": {
						"jobName": test_execution_skipped,
					},
				}, // testNamesToJobNamesToExecutionStatus
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createReportData(tt.args.testNameFilterRegexp, tt.args.testNamesToJobNamesToExecutionStatus); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createReportData() = %v, want %v", got, tt.want)
			}
		})
	}
}

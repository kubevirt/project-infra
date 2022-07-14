package main

import (
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"testing"
)

func Test_writeHTMLReportToOutput(t *testing.T) {
	type args struct {
		htmlReportOutputWriter       io.Writer
		testNames                    []string
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
			writeHTMLReportToOutput(tt.args.htmlReportOutputWriter, tt.args.testNames, tt.args.skippedTests, tt.args.lookedAtJobs, tt.args.testNamesToJobNamesToSkipped, tt.args.err, tt.args.jLog)
		})
	}
}

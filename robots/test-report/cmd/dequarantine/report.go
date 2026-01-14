/*
 * This file is part of the KubeVirt project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * Copyright 2023 Red Hat, Inc.
 *
 */

package dequarantine

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/spf13/cobra"
	testreport "kubevirt.io/project-infra/pkg/test-report"
)

const shortDequarantineReportUsage = "test-report dequarantine report generates a report of the test status for each entry in the quarantined_tests.json"

var dequarantineReportCmd = &cobra.Command{
	Use:   "report",
	Short: shortDequarantineReportUsage,
	Long: shortDequarantineReportUsage + `

The output format is an extended version of the format from 'quarantined_tests.json', added to each record is a
dictionary of test results per test that matches 'Id', ordered by execution time descending.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDequarantineReport()
	},
}

type dequarantineReportOpts struct {
	quarantineFileURL string
	endpoint          string
	startFrom         time.Duration
	jobNamePattern    string
	maxConnsPerHost   int
	dryRun            bool
	outputFile        string
}

var reportJobNamePattern *regexp.Regexp

var dequarantineReportOptions = dequarantineReportOpts{}

func (r *dequarantineReportOpts) Validate() error {
	if r.quarantineFileURL == "" {
		return fmt.Errorf("quarantineFileURL must be set")
	}
	if r.jobNamePattern == "" {
		return fmt.Errorf("jobNamePattern must be set")
	}
	_, err := regexp.Compile(r.jobNamePattern)
	if err != nil {
		return fmt.Errorf("jobNamePattern %q is not a valid regexp", r.jobNamePattern)
	}
	return nil
}

func init() {
	dequarantineReportCmd.PersistentFlags().StringVar(&dequarantineReportOptions.endpoint, "endpoint", testreport.DefaultJenkinsBaseUrl, "jenkins base url")
	dequarantineReportCmd.PersistentFlags().DurationVar(&dequarantineReportOptions.startFrom, "start-from", 10*24*time.Hour, "time period for report")
	dequarantineReportCmd.PersistentFlags().StringVar(&dequarantineReportOptions.quarantineFileURL, "quarantine-file-url", "", "the url to the quarantine file")
	dequarantineReportCmd.PersistentFlags().StringVar(&dequarantineReportOptions.jobNamePattern, "job-name-pattern", "", "the pattern to which all jobs have to match")
	dequarantineReportCmd.PersistentFlags().IntVar(&dequarantineReportOptions.maxConnsPerHost, "max-conns-per-host", 3, "the maximum number of connections that are going to be made")
	dequarantineReportCmd.PersistentFlags().StringVar(&dequarantineReportOptions.outputFile, "output-file", "", "Path to output file, if not given, a temporary file will be used")
	dequarantineReportCmd.PersistentFlags().BoolVar(&dequarantineReportOptions.dryRun, "dry-run", true, "whether to only check what jobs are being considered and then exit")
}

func runDequarantineReport() error {

	err := dequarantineReportOptions.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate command line arguments: %v", err)
	}

	reportJobNamePattern = regexp.MustCompile(dequarantineReportOptions.jobNamePattern)

	client := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost: dequarantineReportOptions.maxConnsPerHost,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	ctx := context.Background()

	logger.Printf("Creating client for %s", dequarantineReportOptions.endpoint)
	jenkins := gojenkins.CreateJenkins(client, dequarantineReportOptions.endpoint)
	_, err = jenkins.Init(ctx)
	if err != nil {
		logger.Fatalf("failed to contact jenkins %s: %v", dequarantineReportOptions.endpoint, err)
	}

	jobNames, err := jenkins.GetAllJobNames(ctx)
	if err != nil {
		logger.Fatalf("failed to get jobs: %v", err)
	}
	jobs, err := testreport.FilterMatchingJobsByJobNamePattern(ctx, jenkins, jobNames, reportJobNamePattern)
	if err != nil {
		logger.Fatalf("failed to filter matching jobs: %v", err)
	}
	var filteredJobNames []string
	for _, job := range jobs {
		filteredJobNames = append(filteredJobNames, job.GetName())
	}
	logger.Infof("jobs that are being considered: %s", strings.Join(filteredJobNames, ", "))
	if dequarantineReportOptions.dryRun {
		logger.Warn("dry-run mode, exiting")
		return nil
	}
	if len(jobs) == 0 {
		logger.Warn("no jobs left, nothing to do")
		return nil
	}

	quarantinedTestEntriesFromFile, err := testreport.FetchDontRunEntriesFromFile(dequarantineReportOptions.quarantineFileURL, client)
	if err != nil {
		logger.Fatalf("failed to filter matching jobs: %v", err)
	}

	startOfReport := time.Now().Add(-1 * dequarantineReportOptions.startFrom)

	quarantinedTestsRunDataValues := generateDequarantineBaseData(jenkins, ctx, jobs, startOfReport, quarantinedTestEntriesFromFile)

	outputFile, err := writeReportData(quarantinedTestsRunDataValues)
	if err != nil {
		return err
	}
	logger.Infof("Report data written to '%s'", outputFile.Name())
	return nil
}

// writeReportData writes the condensed report data into a file
func writeReportData(quarantinedTestsRunDataValues []*quarantinedTestsRunData) (*os.File, error) {
	outputFile, err := createOutputFile(dequarantineReportOptions.outputFile)
	if err != nil {
		return nil, err
	}
	err = json.NewEncoder(outputFile).Encode(quarantinedTestsRunDataValues)
	if err != nil {
		return nil, fmt.Errorf("failed to write report: %v", err)
	}
	return outputFile, nil
}

func createOutputFile(outputFileToCreate string) (outputFile *os.File, err error) {
	if outputFileToCreate == "" {
		outputFile, err = os.CreateTemp("", "quarantined-tests-run-*.json")
		if err != nil {
			return nil, fmt.Errorf("failed to create temporary output file: %v", err)
		}
	} else {
		outputFile, err = os.Create(outputFileToCreate)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file %s: %v", outputFileToCreate, err)
		}
	}
	return outputFile, err
}

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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package execution

import (
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"kubevirt.io/project-infra/pkg/flakefinder"
	testreport "kubevirt.io/project-infra/pkg/test-report"

	"github.com/bndr/gojenkins"
	"github.com/sirupsen/logrus"
	"sigs.k8s.io/yaml"

	"github.com/spf13/cobra"
)

const shortUsage = "test-report execution creates a report in html format to show which tests have been run on what lane"

var executionCmd = &cobra.Command{
	Use:   "execution",
	Short: shortUsage,
	Long: shortUsage + `

It constructs a matrix of test lanes by test names and shows for each test:
* on which lane(s) the test actually was run
* on which lane(s) the test is not supported (taking the information from the 'dont_run_tests.json' configured for the lane
  (see the configurations available)

Tests that are not run on any lane are especially marked in order to emphasize that fact.

Accompanying the html file a json data file is emitted for further consumption.

Note: generating a default report can take a while and will emit a report of enormous size, therefore you can strip down
the output by selecting a configuration that reports over a subset of the data using --config. You also can create your
own configuration file to adjust the report output to your requirements.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runExecutionReport()
	},
}

var logger *logrus.Entry

func ExecutionCmd(rootLogger *logrus.Entry) *cobra.Command {
	logger = rootLogger
	return executionCmd
}

var executionReportFlagOpts = executionReportFlagOptions{}

func init() {
	executionCmd.PersistentFlags().StringVar(&executionReportFlagOpts.endpoint, "endpoint", testreport.DefaultJenkinsBaseUrl, "jenkins base url")
	executionCmd.PersistentFlags().DurationVar(&executionReportFlagOpts.startFrom, "start-from", 14*24*time.Hour, "time period for report")
	executionCmd.PersistentFlags().StringVar(&executionReportFlagOpts.configFile, "config-file", "", "yaml file that contains job names associated with dont_run_tests.json and the job name pattern, if set overrides default-config.yaml")
	var keys []string
	for key := range configs {
		keys = append(keys, key)
	}
	executionCmd.PersistentFlags().StringVar(&executionReportFlagOpts.config, "config", "default", fmt.Sprintf("one of { %s }, chooses one of the default configurations, if set overrides default-config.yaml", strings.Join(keys, ", ")))
	executionCmd.PersistentFlags().StringVar(&executionReportFlagOpts.outputFile, "output-file", "", "Path to output file, if not given, a temporary file will be used")
	executionCmd.PersistentFlags().BoolVar(&executionReportFlagOpts.overwrite, "overwrite", true, "overwrite output file")
	executionCmd.PersistentFlags().BoolVar(&executionReportFlagOpts.dryRun, "dry-run", false, "only check which jobs would be considered, do not create an actual report")
	executionCmd.PersistentFlags().StringVar(&executionReportFlagOpts.exportConfigFilePath, "config-file-export-path", "", "just export selected config as a file, do not create an actual report")
}

type executionReportFlagOptions struct {
	endpoint             string
	startFrom            time.Duration
	configFile           string
	config               string
	outputFile           string
	overwrite            bool
	dryRun               bool
	exportConfigFilePath string
}

func (o *executionReportFlagOptions) Validate() error {
	if o.outputFile == "" {
		outputFile, err := os.CreateTemp("", "test-report-*.html")
		if err != nil {
			return fmt.Errorf("failed to write report: %v", err)
		}
		o.outputFile = outputFile.Name()
	} else {
		err := o.validateOverwrite()
		if err != nil {
			return fmt.Errorf("failed to write report: %v", err)
		}
	}
	if o.configFile != "" {
		if o.exportConfigFilePath != "" {
			return fmt.Errorf("exporting --config-file is a redundant operation")
		}
		_, err := os.Stat(o.configFile)
		if err != nil && errors.Is(err, os.ErrNotExist) {
			return err
		}
		file, err := os.ReadFile(o.configFile)
		if err != nil {
			return err
		}
		err = yaml.Unmarshal(file, &config)
		if err != nil {
			return err
		}
		logger.Printf("Using config file %q", o.configFile)
		configName = "custom"
	} else {
		configBytes, exists := configs[o.config]
		if !exists {
			return fmt.Errorf("config %s not found", o.config)
		}
		logger.Printf("No config file provided, using %s config:/n%s", o.config, string(configBytes))
		configName = o.config
		err := yaml.Unmarshal(configBytes, &config)
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *executionReportFlagOptions) validateOverwrite() error {
	stat, err := os.Stat(o.outputFile)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if stat.IsDir() {
		return fmt.Errorf("failed to write report, file %s is a directory", o.outputFile)
	}
	if err == nil && !o.overwrite {
		return fmt.Errorf("failed to write report, file %s exists", o.outputFile)
	}
	return nil
}

//go:embed test-report.gohtml
var reportTemplate string

//go:embed "default-config.yaml"
var defaultConfigFileContent []byte

//go:embed "compute-config.yaml"
var computeConfigFileContent []byte

//go:embed "network-config.yaml"
var networkConfigFileContent []byte

//go:embed "storage-config.yaml"
var storageConfigFileContent []byte

//go:embed "ssp-config.yaml"
var sspConfigFileContent []byte

var configs = map[string][]byte{
	"default": defaultConfigFileContent,
	"compute": computeConfigFileContent,
	"network": networkConfigFileContent,
	"storage": storageConfigFileContent,
	"ssp":     sspConfigFileContent,
}

var config *testreport.Config
var configName string

func runExecutionReport() error {

	err := executionReportFlagOpts.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate command line arguments: %v", err)
	}

	if executionReportFlagOpts.exportConfigFilePath != "" {
		_, err = os.Stat(executionReportFlagOpts.exportConfigFilePath)
		if err == nil {
			return fmt.Errorf("target file exists, cancelling export to %q", executionReportFlagOpts.exportConfigFilePath)
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}

		err = os.WriteFile(executionReportFlagOpts.exportConfigFilePath, configs[executionReportFlagOpts.config], 0644)
		if err != nil {
			return err
		}
		logger.Infof("config file written to %q, cancelling report creation", executionReportFlagOpts.exportConfigFilePath)
		return nil
	}

	client := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost: config.MaxConnsPerHost,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	jobNamePatternsToTestNameFilterRegexps, err := testreport.CreateJobNamePatternsToTestNameFilterRegexps(config, client)
	if err != nil {
		logger.Fatalf("failed to create test filter regexp: %v", err)
	}

	ctx := context.Background()

	logger.Printf("Creating client for %s", executionReportFlagOpts.endpoint)
	jenkins := gojenkins.CreateJenkins(client, executionReportFlagOpts.endpoint)
	_, err = jenkins.Init(ctx)
	if err != nil {
		logger.Fatalf("failed to contact jenkins %s: %v", executionReportFlagOpts.endpoint, err)
	}

	jobNamePattern := regexp.MustCompile(config.JobNamePattern)

	jobNames, err := jenkins.GetAllJobNames(ctx)
	if err != nil {
		logger.Fatalf("failed to get jobs: %v", err)
	}
	jobs, err := testreport.FilterMatchingJobsByJobNamePattern(ctx, jenkins, jobNames, jobNamePattern)
	if err != nil {
		logger.Fatalf("failed to filter matching jobs: %v", err)
	}
	var filteredJobNames []string
	for _, job := range jobs {
		filteredJobNames = append(filteredJobNames, job.GetName())
	}
	logger.Infof("jobs that are being considered: %s", strings.Join(filteredJobNames, ", "))
	if executionReportFlagOpts.dryRun {
		logger.Warn("dry-run mode, exiting")
		return nil
	}
	if len(jobs) == 0 {
		logger.Warn("no jobs left, nothing to do")
		return nil
	}

	startOfReport := time.Now().Add(-1 * executionReportFlagOpts.startFrom)
	endOfReport := time.Now()

	var jobNamePatternForTestNames *regexp.Regexp
	if config.JobNamePatternForTestNames != "" {
		jobNamePatternForTestNames = regexp.MustCompile(config.JobNamePatternForTestNames)
	}
	testNamesToJobNamesToExecutionStatus := testreport.GetTestNamesToJobNamesToTestExecutions(jobs, startOfReport, ctx, regexp.MustCompile(config.TestNamePattern), jobNamePatternForTestNames)

	data := testreport.CreateReportData(jobNamePatternsToTestNameFilterRegexps, testNamesToJobNamesToExecutionStatus)

	err = writeJsonBaseDataFile(data.TestNamesToJobNamesToSkipped)
	if err != nil {
		logger.Fatalf("failed to write json data file: %v", err)
	}

	data.SetDataRange(startOfReport, endOfReport)
	reportConfig, err := yaml.Marshal(config)
	if err != nil {
		logger.Fatalf("failed to marshall config: %v", err)
	}
	data.SetReportConfig(string(reportConfig))
	data.SetReportConfigName(configName)

	err = writeHTMLReportToOutputFile(data)
	if err != nil {
		logger.Fatalf("failed to write report: %v", err)
	}
	return nil
}

func writeHTMLReportToOutputFile(data testreport.Data) error {
	htmlReportOutputWriter, err := os.OpenFile(executionReportFlagOpts.outputFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to write report %q: %v", executionReportFlagOpts.outputFile, err)
	}
	logger.Printf("Writing html to %q", executionReportFlagOpts.outputFile)
	defer htmlReportOutputWriter.Close()
	err = writeHTMLReportToOutput(data, htmlReportOutputWriter)
	if err != nil {
		return fmt.Errorf("failed to write report: %v", err)
	}
	return nil
}

func writeJsonBaseDataFile(testNamesToJobNamesToExecutionStatus map[string]map[string]int) error {
	bytes, err := json.MarshalIndent(testNamesToJobNamesToExecutionStatus, "", "\t")
	if err != nil {
		return fmt.Errorf("failed to marshall result: %v", err)
	}

	jsonFileName := strings.TrimSuffix(executionReportFlagOpts.outputFile, ".html") + ".json"
	logger.Printf("Writing json to %q", jsonFileName)
	jsonOutputWriter, err := os.OpenFile(jsonFileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil && err != os.ErrNotExist {
		return fmt.Errorf("failed to write report: %v", err)
	}
	defer jsonOutputWriter.Close()

	_, err = jsonOutputWriter.Write(bytes)
	return err
}

func writeHTMLReportToOutput(data testreport.Data, htmlReportOutputWriter io.Writer) error {
	err := flakefinder.WriteTemplateToOutput(reportTemplate, data, htmlReportOutputWriter)
	if err != nil {
		return fmt.Errorf("failed to write report: %v", err)
	}
	return nil
}

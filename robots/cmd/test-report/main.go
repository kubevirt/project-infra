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

package main

import (
	"context"
	"crypto/tls"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bndr/gojenkins"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"kubevirt.io/project-infra/robots/pkg/flakefinder"
	"kubevirt.io/project-infra/robots/pkg/test-report"
	"net/http"
	"os"
	"strings"
	"time"
)

//go:embed test-report.gohtml
var reportTemplate string

var (
	rootCmd *cobra.Command
	opts    options
	logger  *log.Entry
)

func init() {
	rootCmd = &cobra.Command{
		Use:   "test-report",
		Short: "test-report creates a report about which tests have been run on what lane",
		RunE:  runReport,
	}
	opts = flagOptions()
	log.SetLevel(log.Level(opts.logLevel))
	log.SetFormatter(&log.JSONFormatter{})
	logger = log.StandardLogger().WithField("robot", "test-report")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
	os.Exit(0)
}

type options struct {
	endpoint   string
	startFrom  time.Duration
	configFile string
	config     string
	outputFile string
	overwrite  bool
	dryRun     bool
	logLevel   uint32
}

func flagOptions() options {
	opts = options{}
	rootCmd.PersistentFlags().StringVar(&opts.endpoint, "endpoint", test_report.DefaultJenkinsBaseUrl, "jenkins base url")
	rootCmd.PersistentFlags().DurationVar(&opts.startFrom, "start-from", 14*24*time.Hour, "time period for report")
	rootCmd.PersistentFlags().StringVar(&opts.configFile, "config-file", "", "yaml file that contains job names associated with dont_run_tests.json and the job name pattern, if set overrides default-config.yaml")
	rootCmd.PersistentFlags().StringVar(&opts.config, "config", "default", "one of {'default', 'compute', 'storage', 'network'}, chooses one of the default configurations, if set overrides default-config.yaml")
	rootCmd.PersistentFlags().StringVar(&opts.outputFile, "outputFile", "", "Path to output file, if not given, a temporary file will be used")
	rootCmd.PersistentFlags().BoolVar(&opts.overwrite, "overwrite", true, "overwrite output file")
	rootCmd.PersistentFlags().BoolVar(&opts.dryRun, "dry-run", false, "only check which jobs would be considered, do not create an actual report")
	rootCmd.PersistentFlags().Uint32Var(&opts.logLevel, "log-level", 4, "level for logging")
	return opts
}

func (o *options) Validate() error {
	if o.outputFile == "" {
		outputFile, err := os.CreateTemp("", "test-report-*.html")
		if err != nil {
			return fmt.Errorf("failed to write report: %v", err)
		}
		o.outputFile = outputFile.Name()
	} else {
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
	}
	if o.configFile != "" {
		_, err := os.Stat(o.configFile)
		if err != nil && errors.Is(err, os.ErrNotExist) {
			return err
		}
		file, err := ioutil.ReadFile(o.configFile)
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

//go:embed "default-config.yaml"
var defaultConfigFileContent []byte

//go:embed "compute-config.yaml"
var computeConfigFileContent []byte

//go:embed "network-config.yaml"
var networkConfigFileContent []byte

//go:embed "storage-config.yaml"
var storageConfigFileContent []byte

var configs = map[string][]byte{
	"default": defaultConfigFileContent,
	"compute": computeConfigFileContent,
	"network": networkConfigFileContent,
	"storage": storageConfigFileContent,
}

var config *test_report.Config
var configName string

func runReport(cmd *cobra.Command, args []string) error {

	err := opts.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate command line arguments: %v", err)
	}
	jobNamePatternsToTestNameFilterRegexps, err := test_report.CreateJobNamePatternsToTestNameFilterRegexps(config)
	if err != nil {
		logger.Fatalf("failed to create test filter regexp: %v", err)
	}

	ctx := context.Background()

	client := &http.Client{
		Transport: &http.Transport{
			MaxConnsPerHost: config.MaxConnsPerHost,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	logger.Printf("Creating client for %s", opts.endpoint)
	jenkins := gojenkins.CreateJenkins(client, opts.endpoint)
	_, err = jenkins.Init(ctx)
	if err != nil {
		logger.Fatalf("failed to contact jenkins %s: %v", opts.endpoint, err)
	}

	jobNames, err := jenkins.GetAllJobNames(ctx)
	if err != nil {
		logger.Fatalf("failed to get jobs: %v", err)
	}
	jobs, err := test_report.FilterMatchingJobs(ctx, jenkins, jobNames, config)
	if err != nil {
		logger.Fatalf("failed to filter matching jobs: %v", err)
	}
	var filteredJobNames []string
	for _, job := range jobs {
		filteredJobNames = append(filteredJobNames, job.GetName())
	}
	logger.Infof("jobs that are being considered: %s", strings.Join(filteredJobNames, ", "))
	if opts.dryRun {
		logger.Warn("dry-run mode, exiting")
		return nil
	}
	if len(jobs) == 0 {
		logger.Warn("no jobs left, nothing to do")
		return nil
	}

	startOfReport := time.Now().Add(-1 * opts.startFrom)
	endOfReport := time.Now()

	testNamesToJobNamesToExecutionStatus := test_report.GetTestNamesToJobNamesToTestExecutions(jobs, startOfReport, ctx, config)

	err = writeJsonBaseDataFile(testNamesToJobNamesToExecutionStatus)
	if err != nil {
		logger.Fatalf("failed to write json data file: %v", err)
	}

	data := test_report.CreateReportData(jobNamePatternsToTestNameFilterRegexps, testNamesToJobNamesToExecutionStatus)
	data.SetDataRange(startOfReport, endOfReport)
	reportConfig, err := yaml.Marshal(config)
	data.SetReportConfig(string(reportConfig))
	data.SetReportConfigName(configName)

	err = writeHTMLReportToOutputFile(err, data)
	if err != nil {
		logger.Fatalf("failed to write report: %v", err)
	}
	return nil
}

func writeHTMLReportToOutputFile(err error, data test_report.Data) error {
	htmlReportOutputWriter, err := os.OpenFile(opts.outputFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to write report %q: %v", opts.outputFile, err)
	}
	logger.Printf("Writing html to %q", opts.outputFile)
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

	jsonFileName := strings.TrimSuffix(opts.outputFile, ".html") + ".json"
	logger.Printf("Writing json to %q", jsonFileName)
	jsonOutputWriter, err := os.OpenFile(jsonFileName, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil && err != os.ErrNotExist {
		return fmt.Errorf("failed to write report: %v", err)
	}
	defer jsonOutputWriter.Close()

	_, err = jsonOutputWriter.Write(bytes)
	return err
}

func writeHTMLReportToOutput(data test_report.Data, htmlReportOutputWriter io.Writer) error {
	err := flakefinder.WriteTemplateToOutput(reportTemplate, data, htmlReportOutputWriter)
	if err != nil {
		return fmt.Errorf("failed to write report: %v", err)
	}
	return nil
}

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
 * Copyright the KubeVirt Authors.
 *
 */

package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/google/go-github/v28/github"
	"github.com/joshdk/go-junit"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"sigs.k8s.io/prow/pkg/config/secret"
	"sigs.k8s.io/yaml"

	"kubevirt.io/project-infra/pkg/flakefinder"
	ghapi "kubevirt.io/project-infra/pkg/flakefinder/github"
)

const (
	ExecNoData = iota
	ExecSkipped
	ExecRan
	ExecQuarantined
)

//go:embed default-config.yaml
var defaultConfigBytes []byte

//go:embed report.gohtml
var reportTemplate string

type Config struct {
	JobNamePattern  string `yaml:"jobNamePattern"`
	TestNamePattern string `yaml:"testNamePattern"`
}

type ReportData struct {
	BaseURL          string
	Bucket           string
	TestNames        []string
	QuarantinedTests map[string]struct{}
	SkippedTests     map[string]struct{}
	Jobs             []string
	Matrix           map[string]map[string]int
	StatusLabels     map[string]int
	StartOfReport    string
	EndOfReport      string
	Config           string
}

var opts struct {
	token      string
	org        string
	repo       string
	baseBranch string
	bucket     string
	startFrom  time.Duration
	configFile string
	outputFile string
	baseURL    string
	dryRun     bool
}

var rootCmd = &cobra.Command{
	Use:   "test-execution-report",
	Short: "Creates an HTML report showing which tests are run on which presubmit lane",
	RunE:  run,
}

func init() {
	rootCmd.Flags().StringVar(&opts.token, "token", "", "path to GitHub token (required)")
	rootCmd.Flags().StringVar(&opts.org, "org", "kubevirt", "GitHub organization")
	rootCmd.Flags().StringVar(&opts.repo, "repo", "kubevirt", "GitHub repository")
	rootCmd.Flags().StringVar(&opts.baseBranch, "pr-base-branch", "main", "base branch for PR query")
	rootCmd.Flags().StringVar(&opts.bucket, "bucket", flakefinder.BucketName, "GCS bucket name")
	rootCmd.Flags().DurationVar(&opts.startFrom, "start-from", 14*24*time.Hour, "time window for merged PRs")
	rootCmd.Flags().StringVar(&opts.configFile, "config-file", "", "YAML config file (default: embedded config)")
	rootCmd.Flags().StringVar(&opts.outputFile, "output-file", "", "output HTML file path (default: temp file)")
	rootCmd.Flags().StringVar(&opts.baseURL, "base-url", "https://prow.ci.kubevirt.io", "Prow deck base URL for links")
	rootCmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "list matching jobs and exit")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	if opts.token == "" {
		return fmt.Errorf("--token is required")
	}

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %v", err)
	}

	jobPattern := regexp.MustCompile(cfg.JobNamePattern)
	testPattern := regexp.MustCompile(cfg.TestNamePattern)

	if err := secret.Add(opts.token); err != nil {
		return fmt.Errorf("loading token: %v", err)
	}

	ctx := context.Background()

	ghClient := github.NewClient(oauth2.NewClient(ctx, oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: string(secret.GetSecret(opts.token))},
	)))
	query := ghapi.NewQuery(ghClient, opts.org, opts.repo, opts.baseBranch)

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("creating GCS client: %v", err)
	}

	endOfReport := time.Now()
	startOfReport := endOfReport.Add(-opts.startFrom)

	log.Infof("Querying merged PRs from %s to %s", startOfReport.Format(time.RFC3339), endOfReport.Format(time.RFC3339))
	changes, err := query.Query(ctx, startOfReport, endOfReport)
	if err != nil {
		return fmt.Errorf("querying PRs: %v", err)
	}
	log.Infof("Found %d merged PRs", len(changes))

	if len(changes) == 0 {
		log.Warn("no merged PRs found in time window")
		return nil
	}

	var allResults []*flakefinder.JobResult
	repoPath := strings.Join([]string{opts.org, opts.repo}, "/")
	for _, change := range changes {
		results, err := flakefinder.FindUnitTestFiles(ctx, storageClient, opts.bucket, repoPath, change, startOfReport, true)
		if err != nil {
			log.Warnf("failed to load JUnit for PR %d: %v", change.ID(), err)
			continue
		}
		allResults = append(allResults, results...)
	}

	batchResults, err := flakefinder.FindUnitTestFilesForBatchJobs(ctx, storageClient, opts.bucket, nil, changes, startOfReport, endOfReport)
	if err != nil {
		log.Warnf("failed to load batch job JUnit: %v", err)
	}
	allResults = append(allResults, batchResults...)

	// Filter by job name pattern
	var filtered []*flakefinder.JobResult
	for _, r := range allResults {
		if jobPattern.MatchString(r.Job) {
			filtered = append(filtered, r)
		}
	}

	jobSet := map[string]struct{}{}
	for _, r := range filtered {
		jobSet[r.Job] = struct{}{}
	}
	jobs := sortedKeys(jobSet)

	log.Infof("Matched %d jobs: %s", len(jobs), strings.Join(jobs, ", "))
	if opts.dryRun {
		for _, j := range jobs {
			fmt.Println(j)
		}
		return nil
	}
	if len(jobs) == 0 {
		log.Warn("no matching jobs found")
		return nil
	}

	matrix := buildMatrix(filtered, testPattern)

	testNames, skippedTests, quarantinedTests := classifyTests(matrix)

	configBytes, _ := yaml.Marshal(cfg)
	data := ReportData{
		BaseURL:          opts.baseURL,
		Bucket:           opts.bucket,
		TestNames:        testNames,
		QuarantinedTests: quarantinedTests,
		SkippedTests:     skippedTests,
		Jobs:             jobs,
		Matrix:           matrix,
		StatusLabels: map[string]int{
			"ExecNoData":      ExecNoData,
			"ExecSkipped":     ExecSkipped,
			"ExecRan":         ExecRan,
			"ExecQuarantined": ExecQuarantined,
		},
		StartOfReport: startOfReport.Format(time.RFC1123),
		EndOfReport:   endOfReport.Format(time.RFC1123),
		Config:        string(configBytes),
	}

	outputFile, err := resolveOutputFile()
	if err != nil {
		return err
	}

	if err := writeJSONSidecar(outputFile, matrix); err != nil {
		return err
	}

	htmlWriter, err := os.OpenFile(outputFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("opening output file: %v", err)
	}
	defer func() { _ = htmlWriter.Close() }()

	log.Infof("Writing HTML report to %s", outputFile)
	if err := flakefinder.WriteTemplateToOutput(reportTemplate, data, htmlWriter); err != nil {
		return fmt.Errorf("rendering report: %v", err)
	}

	return nil
}

func loadConfig() (*Config, error) {
	var raw []byte
	if opts.configFile != "" {
		var err error
		raw, err = os.ReadFile(opts.configFile)
		if err != nil {
			return nil, err
		}
	} else {
		raw = defaultConfigBytes
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func buildMatrix(results []*flakefinder.JobResult, testPattern *regexp.Regexp) map[string]map[string]int {
	matrix := map[string]map[string]int{}

	for _, r := range results {
		for _, suite := range r.JUnit {
			for _, test := range suite.Tests {
				if !testPattern.MatchString(test.Name) {
					continue
				}
				if _, ok := matrix[test.Name]; !ok {
					matrix[test.Name] = map[string]int{}
				}

				status := execStatus(test)
				cur := matrix[test.Name][r.Job]
				if status > cur {
					matrix[test.Name][r.Job] = status
				}
			}
		}
	}

	return matrix
}

func execStatus(test junit.Test) int {
	if test.Status == junit.StatusSkipped {
		if strings.Contains(test.Name, "[QUARANTINE]") {
			return ExecQuarantined
		}
		return ExecSkipped
	}
	return ExecRan
}

func classifyTests(matrix map[string]map[string]int) (testNames []string, skippedTests, quarantinedTests map[string]struct{}) {
	skippedTests = map[string]struct{}{}
	quarantinedTests = map[string]struct{}{}

	for name, jobs := range matrix {
		testNames = append(testNames, name)

		if strings.Contains(name, "[QUARANTINE]") {
			quarantinedTests[name] = struct{}{}
		}

		ranOnAnyLane := false
		for _, status := range jobs {
			if status == ExecRan {
				ranOnAnyLane = true
				break
			}
		}
		if !ranOnAnyLane {
			skippedTests[name] = struct{}{}
		}
	}

	sort.Strings(testNames)
	return
}

func resolveOutputFile() (string, error) {
	if opts.outputFile != "" {
		return opts.outputFile, nil
	}
	f, err := os.CreateTemp("", "test-execution-report-*.html")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %v", err)
	}
	_ = f.Close()
	return f.Name(), nil
}

func writeJSONSidecar(htmlPath string, matrix map[string]map[string]int) error {
	jsonPath := strings.TrimSuffix(htmlPath, ".html") + ".json"
	data, err := json.MarshalIndent(matrix, "", "\t")
	if err != nil {
		return fmt.Errorf("marshalling JSON: %v", err)
	}
	log.Infof("Writing JSON to %s", jsonPath)
	return os.WriteFile(jsonPath, data, 0644)
}

func sortedKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

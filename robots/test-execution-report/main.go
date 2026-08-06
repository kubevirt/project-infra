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
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	"github.com/joshdk/go-junit"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	"kubevirt.io/project-infra/pkg/flakefinder"
)

const (
	ExecNoData = iota
	ExecSkipped
	ExecRan
	ExecQuarantined
)

var releaseBranchSuffix = regexp.MustCompile(`-\d+\.\d+$`)

//go:embed default-config.yaml
var defaultConfigBytes []byte

//go:embed report.gohtml
var reportTemplate string

type Config struct {
	JobNamePattern         string `yaml:"jobNamePattern"`
	PeriodicJobNamePattern string `yaml:"periodicJobNamePattern"`
	TestNamePattern        string `yaml:"testNamePattern"`
	jobPattern             *regexp.Regexp
	periodicJobPattern     *regexp.Regexp
	testPattern            *regexp.Regexp
}

type ReportData struct {
	BaseURL          string
	Bucket           string
	TestNames        []string
	QuarantinedTests map[string]bool
	SkippedTests     map[string]bool
	Jobs             []string
	PeriodicJobs     map[string]bool
	Matrix           map[string]map[string]int
	StatusLabels     map[string]int
	StartOfReport    string
	EndOfReport      string
	Config           string
}

var opts struct {
	org        string
	repo       string
	bucket     string
	startFrom  time.Duration
	configFile string
	outputFile string
	baseURL    string
	source     string
	dryRun     bool
}

var rootCmd = &cobra.Command{
	Use:   "test-execution-report",
	Short: "Creates an HTML report showing which tests are run on which presubmit and/or periodic lane",
	RunE:  run,
}

func init() {
	rootCmd.Flags().StringVar(&opts.org, "org", "kubevirt", "GitHub organization")
	rootCmd.Flags().StringVar(&opts.repo, "repo", "kubevirt", "GitHub repository")
	rootCmd.Flags().StringVar(&opts.bucket, "bucket", flakefinder.BucketName, "GCS bucket name")
	rootCmd.Flags().DurationVar(&opts.startFrom, "start-from", 14*24*time.Hour, "time window for report data")
	rootCmd.Flags().StringVar(&opts.configFile, "config-file", "", "YAML config file (default: embedded config)")
	rootCmd.Flags().StringVar(&opts.outputFile, "output-file", "", "output HTML file path (default: temp file)")
	rootCmd.Flags().StringVar(&opts.baseURL, "base-url", "https://prow.ci.kubevirt.io", "Prow deck base URL for links")
	rootCmd.Flags().StringVar(&opts.source, "source", "presubmit", "data source: presubmit, periodic, or all")
	rootCmd.Flags().BoolVar(&opts.dryRun, "dry-run", false, "list matching jobs and exit")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	source := opts.source
	if source != "presubmit" && source != "periodic" && source != "all" {
		return fmt.Errorf("--source must be one of: presubmit, periodic, all")
	}

	includePresubmit := source == "presubmit" || source == "all"
	includePeriodic := source == "periodic" || source == "all"

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if includePresubmit && cfg.jobPattern == nil {
		return fmt.Errorf("jobNamePattern is required for presubmit source")
	}
	if includePeriodic && cfg.periodicJobPattern == nil {
		return fmt.Errorf("periodicJobNamePattern is required for periodic source")
	}

	ctx := context.Background()

	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return fmt.Errorf("creating GCS client: %v", err)
	}

	endOfReport := time.Now()
	startOfReport := endOfReport.Add(-opts.startFrom)

	var allResults []*flakefinder.JobResult
	periodicJobs := map[string]bool{}

	if includePresubmit {
		presubmitResults, err := fetchPresubmitResults(ctx, storageClient, cfg, startOfReport, endOfReport)
		if err != nil {
			return err
		}
		allResults = append(allResults, presubmitResults...)
	}

	if includePeriodic {
		periodicResults, err := fetchPeriodicResults(ctx, storageClient, cfg, startOfReport, endOfReport)
		if err != nil {
			return err
		}
		allResults = append(allResults, periodicResults...)
		for _, r := range periodicResults {
			periodicJobs[r.Job] = true
		}
	}

	jobSet := map[string]struct{}{}
	for _, r := range allResults {
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

	matrix := buildMatrix(allResults, cfg.testPattern)

	testNames, skippedTests, quarantinedTests := classifyTests(matrix)

	configBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	data := ReportData{
		BaseURL:          opts.baseURL,
		Bucket:           opts.bucket,
		TestNames:        testNames,
		QuarantinedTests: quarantinedTests,
		SkippedTests:     skippedTests,
		Jobs:             jobs,
		PeriodicJobs:     periodicJobs,
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

func fetchPresubmitResults(ctx context.Context, storageClient *storage.Client, cfg *Config, startOfReport, endOfReport time.Time) ([]*flakefinder.JobResult, error) {
	startPR, err := resolveLatestPR(ctx, storageClient, cfg)
	if err != nil {
		return nil, fmt.Errorf("resolving latest PR number: %v", err)
	}
	log.Infof("Starting presubmit scan from PR %d", startPR)

	repoPrefix := path.Join("pr-logs", "pull", opts.org+"_"+opts.repo)

	var allResults []*flakefinder.JobResult
	consecutiveStale := 0
	const maxConsecutiveStale = 20

	for prNum := startPR; prNum > 0 && consecutiveStale < maxConsecutiveStale; prNum-- {
		prDir := path.Join(repoPrefix, strconv.Itoa(prNum))
		jobDirs, err := flakefinder.ListGcsObjects(ctx, storageClient, opts.bucket, prDir+"/", "/")
		if err != nil {
			log.Warnf("failed to list jobs for PR %d: %v", prNum, err)
			continue
		}
		if len(jobDirs) == 0 {
			continue
		}

		prHasRecentBuild := false
		for _, jobName := range jobDirs {
			if !cfg.jobPattern.MatchString(jobName) {
				continue
			}
			if releaseBranchSuffix.MatchString(jobName) {
				continue
			}
			jobDir := path.Join(prDir, jobName)
			buildDirs, err := flakefinder.ListGcsObjects(ctx, storageClient, opts.bucket, jobDir+"/", "/")
			if err != nil {
				log.Warnf("failed to list builds for PR %d job %s: %v", prNum, jobName, err)
				continue
			}
			builds := flakefinder.SortBuilds(buildDirs)
			if len(builds) == 0 {
				continue
			}

			latestBuild := builds[0]
			buildDir := path.Join(jobDir, strconv.Itoa(latestBuild))
			finishedPath := path.Join(buildDir, "finished.json")

			attrs, err := flakefinder.ReadGcsObjectAttrs(ctx, storageClient, opts.bucket, finishedPath)
			if err == storage.ErrObjectNotExist {
				continue
			} else if err != nil {
				log.Warnf("failed to read finished.json attrs for PR %d job %s build %d: %v", prNum, jobName, latestBuild, err)
				continue
			}
			if attrs.Created.Before(startOfReport) || attrs.Created.After(endOfReport) {
				continue
			}

			prHasRecentBuild = true
			profilePath := path.Join(buildDir, "artifacts", "junit.functest.xml")
			data, err := flakefinder.ReadGcsObject(ctx, storageClient, opts.bucket, profilePath)
			if err == storage.ErrObjectNotExist {
				log.Infof("no junit.functest.xml for PR %d job %s build %d", prNum, jobName, latestBuild)
				continue
			} else if err != nil {
				log.Warnf("failed to read JUnit for PR %d job %s: %v", prNum, jobName, err)
				continue
			}

			report, err := junit.Ingest(data)
			if err != nil {
				log.Warnf("failed to parse JUnit for PR %d job %s: %v", prNum, jobName, err)
				continue
			}
			allResults = append(allResults, &flakefinder.JobResult{Job: jobName, JUnit: report, BuildNumber: latestBuild, PR: prNum})
		}

		if prHasRecentBuild {
			consecutiveStale = 0
		} else {
			consecutiveStale++
		}
	}

	return allResults, nil
}

func resolveLatestPR(ctx context.Context, storageClient *storage.Client, cfg *Config) (int, error) {
	jobDirs, err := flakefinder.ListGcsObjects(ctx, storageClient, opts.bucket, "pr-logs/directory/", "/")
	if err != nil {
		return 0, fmt.Errorf("listing presubmit job directories: %v", err)
	}

	var bestJob string
	var bestBuildID int
	for _, dir := range jobDirs {
		if !cfg.jobPattern.MatchString(dir) {
			continue
		}
		if releaseBranchSuffix.MatchString(dir) {
			continue
		}
		latestBuildPath := path.Join("pr-logs", "directory", dir, "latest-build.txt")
		data, err := flakefinder.ReadGcsObject(ctx, storageClient, opts.bucket, latestBuildPath)
		if err != nil {
			log.Debugf("skipping job %s: cannot read latest-build.txt: %v", dir, err)
			continue
		}
		buildID, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			continue
		}
		if buildID > bestBuildID {
			bestBuildID = buildID
			bestJob = dir
		}
	}
	if bestJob == "" {
		return 0, fmt.Errorf("no active job matching pattern %q found in pr-logs/directory/", cfg.JobNamePattern)
	}
	log.Infof("Using reference job %s (build %d) to find latest PR number", bestJob, bestBuildID)

	aliasPath := path.Join("pr-logs", "directory", bestJob, strconv.Itoa(bestBuildID)+".txt")
	aliasData, err := flakefinder.ReadGcsObject(ctx, storageClient, opts.bucket, aliasPath)
	if err != nil {
		return 0, fmt.Errorf("reading alias file %s: %v", aliasPath, err)
	}

	return parsePRFromAliasPath(string(aliasData), opts.org+"_"+opts.repo)
}

func parsePRFromAliasPath(aliasContent, repoSlug string) (int, error) {
	content := strings.TrimSpace(aliasContent)
	parts := strings.Split(content, "/")
	for i, part := range parts {
		if part == repoSlug && i+1 < len(parts) {
			prNum, err := strconv.Atoi(parts[i+1])
			if err != nil {
				return 0, fmt.Errorf("cannot parse PR number from alias path %q: %v", content, err)
			}
			return prNum, nil
		}
	}
	return 0, fmt.Errorf("repo slug %q not found in alias path %q", repoSlug, content)
}

func fetchPeriodicResults(ctx context.Context, storageClient *storage.Client, cfg *Config, startOfReport, endOfReport time.Time) ([]*flakefinder.JobResult, error) {
	jobDir := "logs"
	log.Infof("Listing periodic jobs from gs://%s/%s/", opts.bucket, jobDir)

	periodicJobDirs, err := flakefinder.ListGcsObjects(ctx, storageClient, opts.bucket, jobDir+"/", "/")
	if err != nil {
		return nil, fmt.Errorf("listing periodic job directories: %v", err)
	}

	var allResults []*flakefinder.JobResult
	for _, dir := range periodicJobDirs {
		if !cfg.periodicJobPattern.MatchString(dir) {
			continue
		}
		log.Infof("Fetching JUnit for periodic job %s", dir)
		results, err := flakefinder.FindUnitTestFilesForPeriodicJob(ctx, storageClient, opts.bucket, []string{jobDir, dir}, startOfReport, endOfReport)
		if err != nil {
			log.Warnf("failed to load JUnit for periodic job %s: %v", dir, err)
			continue
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
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
	var err error
	if err = yaml.UnmarshalStrict(raw, &cfg); err != nil {
		return nil, err
	}
	if cfg.JobNamePattern != "" {
		cfg.jobPattern, err = regexp.Compile(cfg.JobNamePattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regexp for jobNamePattern: %w", err)
		}
	}
	if cfg.PeriodicJobNamePattern != "" {
		cfg.periodicJobPattern, err = regexp.Compile(cfg.PeriodicJobNamePattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regexp for periodicJobNamePattern: %w", err)
		}
	}
	if cfg.TestNamePattern == "" {
		return nil, fmt.Errorf("testNamePattern is required")
	}
	cfg.testPattern, err = regexp.Compile(cfg.TestNamePattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regexp for testNamePattern: %w", err)
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

func classifyTests(matrix map[string]map[string]int) (testNames []string, skippedTests, quarantinedTests map[string]bool) {
	skippedTests = map[string]bool{}
	quarantinedTests = map[string]bool{}

	for name, jobs := range matrix {
		testNames = append(testNames, name)

		if strings.Contains(name, "[QUARANTINE]") {
			quarantinedTests[name] = true
		}

		ranOnAnyLane := false
		for _, status := range jobs {
			if status == ExecRan {
				ranOnAnyLane = true
				break
			}
		}
		if !ranOnAnyLane {
			skippedTests[name] = true
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

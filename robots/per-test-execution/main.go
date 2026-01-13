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
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"

	"cloud.google.com/go/storage"
	"github.com/Masterminds/semver"
	gojunit "github.com/joshdk/go-junit"
	log "github.com/sirupsen/logrus"
	"kubevirt.io/project-infra/pkg/flakefinder"
	"sigs.k8s.io/yaml"
)

const (
	BucketName             = "kubevirt-prow"
	defaultOutputDirectory = "/tmp"
)

var (
	config *Config

	opts options

	//go:embed config.yaml
	defaultConfig []byte

	//go:embed per-test-execution-top-x.gohtml
	perTestExecutionHTMLTemplate string
	htmlTemplate                 *template.Template

	k8sVersionRegex              = regexp.MustCompile(`^[0-9]\.[1-9][0-9]*$`)
	k8sStableReleaseVersionRegex = regexp.MustCompile(`^v([0-9]\.[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
)

type Config struct {
	Lanes []string `yaml:"lanes"`
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.DebugLevel)
	err := yaml.Unmarshal(defaultConfig, &config)
	if err != nil {
		log.Fatal(err)
	}
	htmlTemplate, err = template.New("perTestExecutionTopX").Parse(perTestExecutionHTMLTemplate)
	if err != nil {
		log.Fatal(err)
	}
}

type options struct {
	Months          int
	Days            int
	K8sVersion      string
	ConfigPath      string
	outputDirectory string
}

func (o options) loadDefaults() error {
	if opts.K8sVersion == "" {
		log.Info("loading default stable k8s version")
		resp, err := http.Get("https://dl.k8s.io/release/stable.txt")
		if err != nil {
			return fmt.Errorf("failed to fetch stable k8s version: %v", err)
		}
		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Errorf("failed to close response body: %v", err)
			}
		}()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		k8sStableReleaseVersion := string(body)
		if !k8sStableReleaseVersionRegex.MatchString(k8sStableReleaseVersion) {
			return fmt.Errorf("Kubernetes stable version %q doesn't match regex", k8sStableReleaseVersion)
		}

		// Align the report to k8s version used in current sig-compute-migrations lane
		// since that lane might not be on the latest version, start from the latest and try earlier k8s versions
		exitCounter := 0
		defaultK8sVersion := k8sStableReleaseVersionRegex.FindAllStringSubmatch(k8sStableReleaseVersion, -1)[0][1]
		for {
			migrationsJobURL := fmt.Sprintf("https://prow.ci.kubevirt.io/job-history/gs/kubevirt-prow/logs/periodic-kubevirt-e2e-k8s-%s-sig-compute-migrations", defaultK8sVersion)
			log.Infof("checking whether %q exists", migrationsJobURL)
			head, err := http.Head(migrationsJobURL)
			if err != nil {
				return fmt.Errorf("head request to %q failed: %+v", migrationsJobURL, err)
			}
			if head.StatusCode == 200 {
				break
			}
			exitCounter++
			if exitCounter >= 3 {
				return fmt.Errorf("determining default k8s version for reports failed: no migration lane found, stopped at %q", migrationsJobURL)
			}
			version := semver.MustParse(defaultK8sVersion)
			defaultK8sVersion = fmt.Sprintf("%d.%d", version.Major(), version.Minor()-1)
		}
		opts.K8sVersion = defaultK8sVersion
	}
	return nil
}

func (o options) validate() error {
	if o.Days < 0 {
		return fmt.Errorf("invalid days value")
	}
	if o.Months <= 0 {
		return fmt.Errorf("invalid months value")
	}
	if !k8sVersionRegex.MatchString(o.K8sVersion) {
		return fmt.Errorf("invalid k8s version")
	}
	if o.ConfigPath != "" {
		providedConfig, err := os.ReadFile(o.ConfigPath)
		if err != nil {
			log.Fatal(err)
		}
		err = yaml.Unmarshal(providedConfig, &config)
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("applied configuration from %q\n%s", o.ConfigPath, string(providedConfig))
	} else {
		log.Infof("using default configuration\n%s", string(defaultConfig))
	}
	if o.outputDirectory != defaultOutputDirectory {
		_, err := os.Stat(o.outputDirectory)
		if os.IsNotExist(err) {
			return fmt.Errorf("output directory %q doesn't exist: %v", o.outputDirectory, err)
		} else if err != nil {
			return fmt.Errorf("error on output directory %q: %v", o.outputDirectory, err)
		}
	}
	return nil
}

type TestExecutions struct {
	Name             string
	TotalExecutions  int
	FailedExecutions int
	LatestFailureURL string
}

type TopXTestExecutions struct {
	PerLaneExecutions   map[string][]*TestExecutions
	SortedLinkFilenames []string
	StartOfReport       time.Time
	EndOfReport         time.Time
}

type ByFailuresDescending []*TestExecutions

func (b ByFailuresDescending) Len() int {
	return len(b)
}
func (b ByFailuresDescending) Less(i, j int) bool {
	if b[i].FailedExecutions > b[j].FailedExecutions {
		return true
	}
	if b[i].FailedExecutions == b[j].FailedExecutions &&
		b[i].TotalExecutions > b[j].TotalExecutions {
		return true
	}
	if b[i].FailedExecutions == b[j].FailedExecutions &&
		b[i].TotalExecutions == b[j].TotalExecutions &&
		b[i].Name < b[j].Name {
		return true
	}
	return false
}
func (b ByFailuresDescending) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func main() {
	flag.IntVar(&opts.Months, "months", 3, "determines how many months in the past till today are covered")
	flag.IntVar(&opts.Days, "days", 0, "determines how many days in the past till today are covered (if > 0, used instead of months)")
	flag.StringVar(&opts.K8sVersion, "kubernetes-version", "", "the k8s major.minor version for the target lane, i.e. 1.31")
	flag.StringVar(&opts.ConfigPath, "config-path", "", "path to the config file")
	flag.StringVar(&opts.outputDirectory, "output-directory", defaultOutputDirectory, "path to the output directory - if set other than the default, it will expect it to exist")
	flag.Parse()

	err := opts.loadDefaults()
	if err != nil {
		log.Fatal(err)
	}
	err = opts.validate()
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	currentYear, currentMonth, currentDay := now.Date()
	endOfReport, err := time.Parse(time.DateOnly, fmt.Sprintf("%04d-%02d-%02d", currentYear, int(currentMonth), currentDay))
	if err != nil {
		log.Fatal(err)
	}
	var startOfReport time.Time
	if opts.Days != 0 {
		startOfReport = endOfReport.AddDate(0, 0, -opts.Days)
	} else {
		startOfReport = endOfReport.AddDate(0, -opts.Months, 0)
		startOfReport, err = time.Parse(time.DateOnly, fmt.Sprintf("%s-01", startOfReport.Format("2006-01")))
		if err != nil {
			log.Fatal(err)
		}
	}

	var reportDir string
	if opts.outputDirectory == defaultOutputDirectory {
		reportDir, err = createReportDir(defaultOutputDirectory, startOfReport, endOfReport)
		if err != nil {
			log.Fatal(err)
		}
		log.Debugf("output directory %q created", reportDir)
	} else {
		reportDir = opts.outputDirectory
	}

	log.Infof("Running reports for %s - %s", startOfReport.Format(time.DateOnly), endOfReport.Format(time.DateOnly))

	jobDir := "logs"
	ctx := context.TODO()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	reportFilenames, err := writeReportFiles(ctx, storageClient, startOfReport, endOfReport, jobDir, reportDir)
	if err != nil {
		log.Fatal(err)
	}

	topXLaneTestExecutions, sortedLinkFilenames, err := readTopXLaneTestExecutionsFromReportFiles(reportFilenames, 10)
	if err != nil {
		log.Fatal(err)
	}

	err = writeHTMLReport(startOfReport, endOfReport, topXLaneTestExecutions, sortedLinkFilenames, reportDir)
	if err != nil {
		log.Fatal(err)
	}
}

func writeHTMLReport(startOfReport time.Time, endOfReport time.Time, topXLaneTestExecutions map[string][]*TestExecutions, sortedLinkFilenames []string, reportDir string) error {
	file, err := os.Create(filepath.Join(reportDir, "index.html"))
	if err != nil {
		return err
	}
	defer file.Close()
	err = htmlTemplate.Execute(file, TopXTestExecutions{StartOfReport: startOfReport, EndOfReport: endOfReport, PerLaneExecutions: topXLaneTestExecutions, SortedLinkFilenames: sortedLinkFilenames})
	if err != nil {
		return err
	}
	log.Infof("html file written to %q", file.Name())
	return nil
}

func readTopXLaneTestExecutionsFromReportFiles(reportFilenames []string, topX int) (testExecutions map[string][]*TestExecutions, sortedLinkFilenames []string, err error) {
	topXLaneTestExecutions, err := fetchTopXLaneTestExecutions(reportFilenames, topX)
	if err != nil {
		return nil, nil, err
	}

	for linkFilename := range topXLaneTestExecutions {
		sortedLinkFilenames = append(sortedLinkFilenames, linkFilename)
	}
	sort.Slice(sortedLinkFilenames, func(i, j int) bool {
		i1, j1 := topXLaneTestExecutions[sortedLinkFilenames[i]], topXLaneTestExecutions[sortedLinkFilenames[j]]
		calculatePercentage := func(testExecutions []*TestExecutions) (failureRate float32) {
			total, failed := 0, 0
			for _, perTestExecution := range testExecutions {
				total += perTestExecution.TotalExecutions
				failed += perTestExecution.FailedExecutions
			}
			if total == 0 {
				return 0
			}
			return float32(failed) / float32(total)
		}
		return calculatePercentage(i1) > calculatePercentage(j1)
	})

	log.Debugf("test executions fetched")
	return topXLaneTestExecutions, sortedLinkFilenames, nil
}

func fetchTopXLaneTestExecutions(reportFilenames []string, topX int) (map[string][]*TestExecutions, error) {
	topXLaneTestExecutions := make(map[string][]*TestExecutions)
	for _, filename := range reportFilenames {
		log.Debugf("Reading top %d entries of generated file %q to create top x list", topX, filename)
		openFile, err := os.OpenFile(filename, os.O_RDONLY, 0666)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %q: %v", filename, err)
		}
		defer openFile.Close()
		csvReader := csv.NewReader(openFile)
		linkFilename := filepath.Base(filename)
		testExecutionsPerLane := []*TestExecutions{}
		var headers []string
		for i := 0; i < topX+1; i++ {
			record, err := csvReader.Read()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return nil, fmt.Errorf("failed to read file %q: %v", filename, err)
			}
			if i == 0 {
				// store headers for lookup of link to latest failure
				headers = record
				continue
			}
			atoi, err := strconv.Atoi(record[1])
			if err != nil {
				return nil, fmt.Errorf("failed to convert value %q: %v", atoi, err)
			}
			totalExecutions := atoi
			atoi, err = strconv.Atoi(record[2])
			if err != nil {
				return nil, fmt.Errorf("failed to convert value %q: %v", atoi, err)
			}
			failedExecutions := atoi

			var failureLink string
			if failedExecutions > 0 {
				// fetch link to latest failure - search for first 'f' character from end to start, since the values
				// are stored chronologically
				for i := len(headers) - 1; i > 0; i-- {
					if record[i] == "f" {
						failureLink = headers[i]
						break
					}
				}
			}

			topXTestExecution := &TestExecutions{
				Name:             record[0],
				TotalExecutions:  totalExecutions,
				FailedExecutions: failedExecutions,
				LatestFailureURL: failureLink,
			}
			testExecutionsPerLane = append(testExecutionsPerLane, topXTestExecution)

		}
		topXLaneTestExecutions[linkFilename] = testExecutionsPerLane
	}
	return topXLaneTestExecutions, nil
}

type writeReportFileResult struct {
	reportFileName string
	err            error
}

func writeReportFiles(ctx context.Context, storageClient *storage.Client, startOfReport time.Time, endOfReport time.Time, jobDir string, reportDir string) ([]string, error) {
	log.Debugf("writing report files for lanes: %v", config.Lanes)

	writeReportFileResults := make(chan writeReportFileResult)
	go doWriteReportFiles(ctx, storageClient, startOfReport, endOfReport, jobDir, writeReportFileResults, reportDir)

	var fileNames []string
	for result := range writeReportFileResults {
		if result.err != nil {
			return nil, result.err
		}
		fileNames = append(fileNames, result.reportFileName)
	}

	log.Debugf("report files: %v", fileNames)
	return fileNames, nil
}

func doWriteReportFiles(ctx context.Context, storageClient *storage.Client, startOfReport time.Time, endOfReport time.Time, jobDir string, writeReportFileResults chan writeReportFileResult, reportDir string) {
	defer close(writeReportFileResults)

	var wg sync.WaitGroup
	wg.Add(len(config.Lanes))
	for _, periodicJobDirPattern := range config.Lanes {
		periodicJobDir := fmt.Sprintf(periodicJobDirPattern, opts.K8sVersion)
		go writeReportFile(&wg, ctx, storageClient, startOfReport, endOfReport, jobDir, periodicJobDir, writeReportFileResults, reportDir)
	}
	wg.Wait()
}

func writeReportFile(wg *sync.WaitGroup, ctx context.Context, storageClient *storage.Client, startOfReport time.Time, endOfReport time.Time, jobDir string, periodicJobDir string, writeReportFileResults chan writeReportFileResult, reportDir string) {
	defer wg.Done()
	log.Debugf("writing file for %q", periodicJobDir)
	results, err := flakefinder.FindUnitTestFilesForPeriodicJob(ctx, storageClient, BucketName, []string{jobDir, periodicJobDir}, startOfReport, endOfReport)
	if err != nil {
		writeReportFileResults <- writeReportFileResult{err: fmt.Errorf("failed to load periodicJobDirs for %v: %v", fmt.Sprintf("%s*", periodicJobDir), fmt.Errorf("error listing gcs objects: %v", err))}
		return
	}

	buildNumbers, perBuildTestExecutions, allTestExecutions := condenseToTestExecutions(periodicJobDir, results)

	log.Debugf("sorting results for %s", periodicJobDir)
	sortedTestExecutions := make([]*TestExecutions, 0, len(allTestExecutions))
	for _, testExecution := range allTestExecutions {
		sortedTestExecutions = append(sortedTestExecutions, testExecution)
	}
	sort.Sort(ByFailuresDescending(sortedTestExecutions))
	sort.Ints(buildNumbers)

	log.Debugf("creating file for %s", periodicJobDir)
	file, err := os.Create(filepath.Join(reportDir, fmt.Sprintf("%s.csv", periodicJobDir)))
	if err != nil {
		writeReportFileResults <- writeReportFileResult{err: err}
		return
	}
	writer := csv.NewWriter(file)
	headers := []string{"test-name", "number-of-executions", "number-of-failures"}
	for _, buildNumber := range buildNumbers {
		headers = append(headers, fmt.Sprintf("https://prow.ci.kubevirt.io/view/gcs/kubevirt-prow/logs/%s/%d", periodicJobDir, buildNumber))
	}
	err = writer.Write(headers)
	if err != nil {
		writeReportFileResults <- writeReportFileResult{err: err}
		return
	}
	reportFileName := file.Name()
	log.Debugf("writing values to file %q", reportFileName)
	for _, testExecution := range sortedTestExecutions {
		values := []string{testExecution.Name, strconv.Itoa(testExecution.TotalExecutions), strconv.Itoa(testExecution.FailedExecutions)}
		for _, buildNumber := range buildNumbers {
			value := ""
			if executionStatus, statusExists := perBuildTestExecutions[testExecution.Name][buildNumber]; statusExists {
				value = string(executionStatus)
			}
			values = append(values, value)
		}
		err = writer.Write(values)
		if err != nil {
			writeReportFileResults <- writeReportFileResult{err: err}
			return
		}
	}
	log.Debugf("flushing file %q", reportFileName)
	writer.Flush()
	err = writer.Error()
	if err != nil {
		writeReportFileResults <- writeReportFileResult{err: err}
		return
	}
	err = file.Close()
	if err != nil {
		writeReportFileResults <- writeReportFileResult{err: err}
		return
	}

	log.Debugf("report for %q written to %q", periodicJobDir, reportFileName)
	writeReportFileResults <- writeReportFileResult{reportFileName, err}
}

func condenseToTestExecutions(periodicJobDir string, results []*flakefinder.JobResult) ([]int, map[string]map[int]rune, map[string]*TestExecutions) {
	log.Debugf("iterating over results for %s", periodicJobDir)
	buildNumbers := make([]int, 0, len(results))
	perBuildTestExecutions := make(map[string]map[int]rune)
	allTestExecutions := make(map[string]*TestExecutions)
	for _, result := range results {
		for _, junit := range result.JUnit {
			buildNumbers = append(buildNumbers, result.BuildNumber)
			for _, test := range junit.Tests {
				testName := flakefinder.NormalizeTestName(test.Name)

				if _, exists := perBuildTestExecutions[testName]; !exists {
					perBuildTestExecutions[testName] = make(map[int]rune)
				}
				r, _ := utf8.DecodeRuneInString(string(test.Status))
				perBuildTestExecutions[testName][result.BuildNumber] = r

				testExecutionRecord := &TestExecutions{
					Name:             testName,
					TotalExecutions:  0,
					FailedExecutions: 0,
				}
				switch test.Status {
				case gojunit.StatusFailed, gojunit.StatusError:
					if _, exists := allTestExecutions[testName]; !exists {
						allTestExecutions[testName] = testExecutionRecord
					}
					testExecutionRecord = allTestExecutions[testName]
					testExecutionRecord.FailedExecutions = testExecutionRecord.FailedExecutions + 1
					testExecutionRecord.TotalExecutions = testExecutionRecord.TotalExecutions + 1
				case gojunit.StatusPassed:
					if _, exists := allTestExecutions[testName]; !exists {
						allTestExecutions[testName] = testExecutionRecord
					}
					testExecutionRecord = allTestExecutions[testName]
					testExecutionRecord.TotalExecutions = testExecutionRecord.TotalExecutions + 1
				default:
					// NOOP
				}
			}
		}
	}
	return buildNumbers, perBuildTestExecutions, allTestExecutions
}

func createReportDir(basedir string, startOfReport, endOfReport time.Time) (string, error) {
	reportDirName := filepath.Join(basedir, fmt.Sprintf(
		"per-test-execution-%s-%s",
		startOfReport.Format("2006-01"),
		endOfReport.Add(-1*24*time.Hour).Format("2006-01")))
	return reportDirName, os.Mkdir(reportDirName, 0777)
}

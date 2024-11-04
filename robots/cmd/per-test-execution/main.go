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
	"cloud.google.com/go/storage"
	"context"
	_ "embed"
	"encoding/csv"
	"flag"
	"fmt"
	junit2 "github.com/joshdk/go-junit"
	log "github.com/sirupsen/logrus"
	"io"
	"kubevirt.io/project-infra/robots/pkg/flakefinder"
	"net/http"
	"os"
	"regexp"
	"sigs.k8s.io/yaml"
	"sort"
	"strconv"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	BucketName = "kubevirt-prow"
)

var (
	config *Config

	opts options

	//go:embed config.yaml
	defaultConfig []byte

	k8sVersionRegex              = regexp.MustCompile(`^[0-9]\.[1-9][0-9]*$`)
	k8sStableReleaseVersionRegex = regexp.MustCompile(`^v([0-9]\.[1-9][0-9]*)\.[1-9][0-9]*$`)
)

type Config struct {
	Lanes []string `yaml:"lanes"`
}

func init() {
	err := yaml.Unmarshal(defaultConfig, &config)
	if err != nil {
		log.Fatal(err)
	}
}

type options struct {
	Months     int
	K8sVersion string
	ConfigPath string
}

func (o options) loadDefaults() error {
	if opts.K8sVersion == "" {
		log.Info("loading default stable k8s version")
		resp, err := http.Get("https://dl.k8s.io/release/stable.txt")
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		k8sStableReleaseVersion := string(body)
		if !k8sStableReleaseVersionRegex.MatchString(k8sStableReleaseVersion) {
			return fmt.Errorf("Kubernetes stable version %q doesn't match regex", k8sStableReleaseVersion)
		}
		opts.K8sVersion = k8sStableReleaseVersionRegex.FindAllStringSubmatch(k8sStableReleaseVersion, -1)[0][1]
	}
	return nil
}

func (o options) validate() error {
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
	return nil
}

type testExecutions struct {
	Name             string
	TotalExecutions  int
	FailedExecutions int
}

type ByFailuresDescending []*testExecutions

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
	flag.IntVar(&opts.Months, "months", 3, "determines how much months in the past till today are covered")
	flag.StringVar(&opts.K8sVersion, "kubernetes-version", "", "the k8s major.minor version for the target lane, i.e. 1.31")
	flag.StringVar(&opts.ConfigPath, "config-path", "", "path to the config file")
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
	startOfReport := endOfReport.AddDate(0, -opts.Months, 0)
	startOfReport, err = time.Parse(time.DateOnly, fmt.Sprintf("%s-01", startOfReport.Format("2006-01")))
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Running reports for %s - %s", startOfReport.Format(time.DateOnly), endOfReport.Format(time.DateOnly))

	jobDir := "logs"
	ctx := context.TODO()
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	var wg sync.WaitGroup
	wg.Add(len(config.Lanes))
	for _, periodicJobDirPattern := range config.Lanes {
		periodicJobDir := fmt.Sprintf(periodicJobDirPattern, opts.K8sVersion)
		go func(wg *sync.WaitGroup, ctx context.Context, periodicJobDir string, storageClient *storage.Client, opts options) {
			defer wg.Done()
			log.Infoln("starting report for ", periodicJobDir)
			results, err := flakefinder.FindUnitTestFilesForPeriodicJob(ctx, storageClient, BucketName, []string{jobDir, periodicJobDir}, startOfReport, endOfReport)
			if err != nil {
				log.Printf("failed to load periodicJobDirs for %v: %v", fmt.Sprintf("%s*", periodicJobDir), fmt.Errorf("error listing gcs objects: %v", err))
			}

			buildNumbers := make([]int, 0, len(results))
			perBuildTestExecutions := make(map[string]map[int]rune)
			allTestExecutions := make(map[string]*testExecutions)
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

						testExecutionRecord := &testExecutions{
							Name:             testName,
							TotalExecutions:  0,
							FailedExecutions: 0,
						}
						switch test.Status {
						case junit2.StatusFailed, junit2.StatusError:
							if _, exists := allTestExecutions[testName]; !exists {
								allTestExecutions[testName] = testExecutionRecord
							}
							testExecutionRecord = allTestExecutions[testName]
							testExecutionRecord.FailedExecutions = testExecutionRecord.FailedExecutions + 1
							testExecutionRecord.TotalExecutions = testExecutionRecord.TotalExecutions + 1
						case junit2.StatusPassed:
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

			sortedTestExecutions := make([]*testExecutions, 0, len(allTestExecutions))
			for _, testExecution := range allTestExecutions {
				sortedTestExecutions = append(sortedTestExecutions, testExecution)
			}
			sort.Sort(ByFailuresDescending(sortedTestExecutions))
			sort.Ints(buildNumbers)

			fileName := fmt.Sprintf(
				"per-test-execution-%s-%s-%s-*.csv",
				startOfReport.Format("2006-01"),
				endOfReport.Add(-1*24*time.Hour).Format("2006-01"),
				periodicJobDir)
			file, err := os.CreateTemp("/tmp", fileName)
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()
			writer := csv.NewWriter(file)
			headers := []string{"test-name", "number-of-executions", "number-of-failures"}
			for _, buildNumber := range buildNumbers {
				headers = append(headers, fmt.Sprintf("https://prow.ci.kubevirt.io/view/gcs/kubevirt-prow/logs/%s/%d", periodicJobDir, buildNumber))
			}
			err = writer.Write(headers)
			if err != nil {
				log.Fatal(err)
			}
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
					log.Fatal(err)
				}
			}
			writer.Flush()
			log.Infoln("Execution data written to ", file.Name())
		}(&wg, ctx, periodicJobDir, storageClient, opts)
	}
	wg.Wait()
}

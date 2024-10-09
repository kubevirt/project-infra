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
	"kubevirt.io/project-infra/robots/pkg/flakefinder"
	"os"
	"regexp"
	"sigs.k8s.io/yaml"
	"sort"
	"strconv"
	"sync"
	"time"
)

const (
	BucketName = "kubevirt-prow"
)

var (
	config *Config

	opts options

	//go:embed config.yaml
	defaultConfig []byte

	k8sVersionRegex = regexp.MustCompile(`^[0-9]\.[1-9][0-9]+$`)
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

func main() {
	flag.IntVar(&opts.Months, "months", 1, "determines how much months in the past till today are covered")
	flag.StringVar(&opts.K8sVersion, "kubernetes-version", "1.31", "string defining the k8s version that has the most recent lane")
	flag.StringVar(&opts.ConfigPath, "config-path", "", "path to the config file")
	flag.Parse()

	err := opts.validate()
	if err != nil {
		log.Fatal(err)
	}

	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	endOfReport, err := time.Parse(time.DateOnly, fmt.Sprintf("%04d-%02d-01", currentYear, int(currentMonth)))
	if err != nil {
		log.Fatal(err)
	}
	startOfReport := endOfReport.AddDate(0, -opts.Months, 0)
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

			allTestExecutions := make(map[string]*testExecutions)
			for _, result := range results {
				for _, junit := range result.JUnit {
					for _, test := range junit.Tests {
						testName := flakefinder.NormalizeTestName(test.Name)
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

			testNames := make([]string, 0, len(allTestExecutions))
			for testName := range allTestExecutions {
				testNames = append(testNames, testName)
			}
			sort.Strings(testNames)

			file, err := os.CreateTemp("/tmp", fmt.Sprintf("per-test-execution-%s-%s-%s-*.csv", startOfReport.Format("2006-01"), endOfReport.Add(-1*24*time.Hour).Format("2006-01"), periodicJobDir))
			if err != nil {
				log.Fatal(err)
			}
			defer file.Close()
			writer := csv.NewWriter(file)
			writer.Write([]string{"test-name", "number-of-executions", "number-of-failures"})
			for _, testName := range testNames {
				testExecution, exists := allTestExecutions[testName]
				if !exists {
					log.Fatal("test %s doesn't exist", testName)
				}
				err := writer.Write([]string{testExecution.Name, strconv.Itoa(testExecution.TotalExecutions), strconv.Itoa(testExecution.FailedExecutions)})
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

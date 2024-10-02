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
	"fmt"
	junit2 "github.com/joshdk/go-junit"
	log "github.com/sirupsen/logrus"
	"kubevirt.io/project-infra/robots/pkg/flakefinder"
	"os"
	"sigs.k8s.io/yaml"
	"sort"
	"strconv"
	"time"
)

const (
	BucketName = "kubevirt-prow"
)

var (
	config *Config
	//go:embed config.yaml
	configContent []byte
)

type Config struct {
	Lanes []string `yaml:"lanes"`
}

func init() {
	err := yaml.Unmarshal(configContent, &config)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	now := time.Now()
	currentYear, currentMonth, _ := now.Date()
	endOfReport, err := time.Parse(time.DateOnly, fmt.Sprintf("%04d-%02d-01", currentYear, int(currentMonth)))
	if err != nil {
		log.Fatal(err)
	}
	startOfReport, err := time.Parse(time.DateOnly, fmt.Sprintf("%04d-%02d-01", currentYear, int(currentMonth)-1))
	//startOfReport := endOfReport.Add(-7 * 24 * time.Hour)
	if err != nil {
		log.Fatal(err)
	}

	jobDir := "logs"
	ctx := context.TODO()
	storageClient, err := storage.NewClient(ctx)

	for _, periodicJobDir := range config.Lanes {
		results, err := flakefinder.FindUnitTestFilesForPeriodicJob(ctx, storageClient, BucketName, []string{jobDir, periodicJobDir}, startOfReport, endOfReport)
		if err != nil {
			log.Printf("failed to load periodicJobDirs for %v: %v", fmt.Sprintf("%s*", periodicJobDir), fmt.Errorf("error listing gcs objects: %v", err))
		}

		type TestExecutions struct {
			Name             string
			TotalExecutions  int
			FailedExecutions int
		}

		allTestExecutions := make(map[string]*TestExecutions)
		testNames := []string{}
		for _, result := range results {
			for _, junit := range result.JUnit {
				for _, test := range junit.Tests {
					testExecutionRecord := &TestExecutions{
						Name:             test.Name,
						TotalExecutions:  0,
						FailedExecutions: 0,
					}
					switch test.Status {
					case junit2.StatusFailed, junit2.StatusError:
						if _, exists := allTestExecutions[test.Name]; !exists {
							allTestExecutions[test.Name] = testExecutionRecord
							testNames = append(testNames, test.Name)
						}
						testExecutionRecord = allTestExecutions[test.Name]
						testExecutionRecord.FailedExecutions = testExecutionRecord.FailedExecutions + 1
						testExecutionRecord.TotalExecutions = testExecutionRecord.TotalExecutions + 1
					case junit2.StatusPassed:
						if _, exists := allTestExecutions[test.Name]; !exists {
							allTestExecutions[test.Name] = testExecutionRecord
							testNames = append(testNames, test.Name)
						}
						testExecutionRecord = allTestExecutions[test.Name]
						testExecutionRecord.TotalExecutions = testExecutionRecord.TotalExecutions + 1
					default:
						// NOOP
					}
				}
			}
		}
		sort.Strings(testNames)

		file, err := os.CreateTemp("/tmp", fmt.Sprintf("per-test-execution-%s-%s-*.csv", startOfReport.Format("2006-01"), periodicJobDir))
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
		log.Infoln("Execution data written to ", file.Name())
	}
}

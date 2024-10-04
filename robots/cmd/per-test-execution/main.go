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
	"time"
)

const (
	BucketName = "kubevirt-prow"
)

var (
	config *Config

	opts options

	//go:embed config.yaml
	configContent []byte

	k8sVersionRegex = regexp.MustCompile(`^[0-9]\.[1-9][0-9]+$`)
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

type options struct {
	Months     int
	K8sVersion string
}

func (o options) validate() error {
	if o.Months <= 0 {
		return fmt.Errorf("invalid months value")
	}
	if !k8sVersionRegex.MatchString(o.K8sVersion) {
		return fmt.Errorf("invalid k8s version")
	}
	return nil
}

func main() {
	flag.IntVar(&opts.Months, "months", 1, "determines how much days in the past till today are covered")
	flag.StringVar(&opts.K8sVersion, "kubernetes-version", "1.31", "string defining the k8s version that has the most recent lane")
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
	startOfReport, err := time.Parse(time.DateOnly, fmt.Sprintf("%04d-%02d-01", currentYear, int(currentMonth)-opts.Months))
	//startOfReport := endOfReport.Add(-7 * 24 * time.Hour)
	if err != nil {
		log.Fatal(err)
	}

	jobDir := "logs"
	ctx := context.TODO()
	storageClient, err := storage.NewClient(ctx)

	for _, periodicJobDirPattern := range config.Lanes {
		periodicJobDir := fmt.Sprintf(periodicJobDirPattern, opts.K8sVersion)
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
					testName := flakefinder.NormalizeTestName(test.Name)
					testExecutionRecord := &TestExecutions{
						Name:             testName,
						TotalExecutions:  0,
						FailedExecutions: 0,
					}
					switch test.Status {
					case junit2.StatusFailed, junit2.StatusError:
						if _, exists := allTestExecutions[testName]; !exists {
							allTestExecutions[testName] = testExecutionRecord
							testNames = append(testNames, testName)
						}
						testExecutionRecord = allTestExecutions[testName]
						testExecutionRecord.FailedExecutions = testExecutionRecord.FailedExecutions + 1
						testExecutionRecord.TotalExecutions = testExecutionRecord.TotalExecutions + 1
					case junit2.StatusPassed:
						if _, exists := allTestExecutions[testName]; !exists {
							allTestExecutions[testName] = testExecutionRecord
							testNames = append(testNames, testName)
						}
						testExecutionRecord = allTestExecutions[testName]
						testExecutionRecord.TotalExecutions = testExecutionRecord.TotalExecutions + 1
					default:
						// NOOP
					}
				}
			}
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
		log.Infoln("Execution data written to ", file.Name())
	}
}

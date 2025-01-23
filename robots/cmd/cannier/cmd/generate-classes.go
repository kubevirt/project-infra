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

package cmd

import (
	"encoding/csv"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"io"
	"kubevirt.io/project-infra/robots/pkg/cannier"
	"net/http"
	"strconv"
)

const sourceDataURL = "https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-compute.csv"

type TestDescriptor struct {
	Name  string
	Label cannier.TestLabel
}

// generateClassesCmd represents the classes command
var generateClassesCmd = &cobra.Command{
	Use:   "classes",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// generate classes for categorization of the model for later usage
		var records [][]string
		for _, url := range perTestExecutionCSVLinks {
			sourceRecords, err := ExtractSourceRecords(url)
			if err != nil {
				return err
			}
			records = append(records, sourceRecords...)
		}

		descriptors, err := DescriptorsFromRecords(records)
		if err != nil {
			return err
		}

		counters := make(map[cannier.TestLabel]int)
		for _, descriptor := range descriptors {
			if _, ok := counters[descriptor.Label]; !ok {
				counters[descriptor.Label] = 0
			}
			counters[descriptor.Label]++
		}

		for testLabel, counter := range counters {
			log.Infof("%d: %d", testLabel, counter)
		}

		return nil
	},
}

func ExtractSourceRecords(sourceDataURL string) ([][]string, error) {
	resp, err := http.Get(sourceDataURL)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error(err)
		}
	}(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("http error %q while retrieving source doc %q received", resp.Status, sourceDataURL)
	}
	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}
	records = records[1:]
	return records, nil
}

func DescriptorsFromRecords(records [][]string) ([]TestDescriptor, error) {
	var descriptors []TestDescriptor
	for _, record := range records {
		testName, numberOfExecutionsString, numberOfFailuresString := record[0], record[1], record[2]
		numberOfExecutions, err := strconv.Atoi(numberOfExecutionsString)
		if err != nil {
			return nil, err
		}
		numberOfFailures, err := strconv.Atoi(numberOfFailuresString)
		if err != nil {
			return nil, err
		}
		var failureRateInPercent float64
		if numberOfExecutions != 0 {
			failureRateInPercent = (float64(numberOfFailures) / float64(numberOfExecutions)) * 100.0
		}
		descriptor := TestDescriptor{
			Name: testName,
		}
		switch {
		case failureRateInPercent == 0:
			descriptor.Label = cannier.MODEL_CLASS_STABLE
		case failureRateInPercent < 100.0:
			descriptor.Label = cannier.MODEL_CLASS_FLAKY
		default:
			descriptor.Label = cannier.MODEL_CLASS_UNSTABLE
		}
		descriptors = append(descriptors, descriptor)
	}
	return descriptors, nil
}

func init() {
	generateCmd.AddCommand(generateClassesCmd)
	generateClassesCmd.Flags().StringP("source-data-url", "u", sourceDataURL, "url for the source data document from per-test-execution in csv format")
}

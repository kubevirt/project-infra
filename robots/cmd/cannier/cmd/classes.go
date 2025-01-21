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
	"net/http"
	"strconv"
)

// classesCmd represents the classes command
var classesCmd = &cobra.Command{
	Use:   "classes",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// generate some classes for categorization of the model for later usage
		resp, err := http.Get("https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-compute.csv")
		if err != nil {
			return err
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Error(err)
			}
		}(resp.Body)
		if resp.StatusCode != 200 {
			return fmt.Errorf("http error %q while retrieving source doc %q received", resp.Status)
		}
		reader := csv.NewReader(resp.Body)
		records, err := reader.ReadAll()
		if err != nil {
			return err
		}

		binRanges := []float64{0, 0.625, 1.25, 2.5, 5.0, 10.0, 25.0, 50.0, 100.0}
		buckets := make([][]float64, len(binRanges), len(binRanges))
		counters := make([]int, len(binRanges), len(binRanges))

		for i, record := range records {
			if i == 0 {
				continue
			}
			testName, numberOfExecutionsString, numberOfFailuresString := record[0], record[1], record[2]
			numberOfExecutions, err := strconv.Atoi(numberOfExecutionsString)
			if err != nil {
				return err
			}
			numberOfFailures, err := strconv.Atoi(numberOfFailuresString)
			if err != nil {
				return err
			}
			var failureRateInPercent float64
			if numberOfExecutions != 0 {
				failureRateInPercent = (float64(numberOfFailures) / float64(numberOfExecutions)) * 100.0
			}
			log.Debugf("%f <- %q", failureRateInPercent, testName)
			for i, binMax := range binRanges {
				if binMax < failureRateInPercent {
					continue
				}
				counters[i]++
				buckets[i] = append(buckets[i], failureRateInPercent)
				break
			}
		}

		for i, binMax := range binRanges {
			log.Infof("%f: %d, %v", binMax, counters[i], buckets[i])
		}

		return nil
	},
}

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	log.SetLevel(log.DebugLevel)

	generateCmd.AddCommand(classesCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// classesCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// classesCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

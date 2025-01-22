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
	"math/rand"
	osexec "os/exec"

	"github.com/malaschitz/randomForest"
	"github.com/spf13/cobra"
)

var perTestExecutionCSVLinks = []string{
	"https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-compute.csv",
	"https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-storage.csv",
	"https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-compute-migrations.csv",
	"https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-operator.csv",
	"https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-network.csv",
}

var defaultKubeVirtSourceCodeTestPath = "../kubevirt/tests"

// modelCmd represents the model command
var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "model generates the random forest model from a set of test cases",
	Long: `model generates the random forest model from a set of test cases

It determines categories from the set of data available via per-test-execution, 
then uses the test names to extract the feature vectors, and then creates the 
dataset for the model. It finally trains the model and stores the model as a 
serialized blob to given path.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// fetch tests from csv
		descriptorsByName := make(map[string]TestDescriptor)
		for _, url := range perTestExecutionCSVLinks {
			records, err := ExtractSourceRecords(url)
			if err != nil {
				return err
			}
			descriptors, err := DescriptorsFromRecords(records)
			if err != nil {
				return err
			}
			for _, descriptor := range descriptors {
				descriptorsByName[descriptor.Name] = descriptor
				// TODO: attach file
				osexec.Command("git", "grep", "-E")
			}
		}

		// 3.) generate model from data
		xData := [][]float64{}
		yData := []int{}
		for i := 0; i < 1000; i++ {
			x := []float64{rand.Float64(), rand.Float64(), rand.Float64(), rand.Float64()}
			y := int(x[0] + x[1] + x[2] + x[3])
			xData = append(xData, x)
			yData = append(yData, y)
		}
		forest := randomforest.Forest{}
		forest.Data = randomforest.ForestData{X: xData, Class: yData}
		forest.Train(1000)

		// 4.) store model to disk via serialization
		// https://stackoverflow.com/questions/51300011/how-to-write-a-struct-as-binary-data-to-a-file-in-golang
		return nil
	},
}

func init() {
	generateCmd.AddCommand(modelCmd)

	modelCmd.Flags().StringP("output-path", "o", "/tmp/kubevirt-cannier.bin", "Help message for toggle")
}

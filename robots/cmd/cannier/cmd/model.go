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
	"github.com/malaschitz/randomForest"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"kubevirt.io/project-infra/robots/pkg/flake-heuristic/cannier"
	"kubevirt.io/project-infra/robots/pkg/ginkgo"
)

var perTestExecutionCSVLinks = []string{
	"https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-compute.csv",
	"https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-storage.csv",
	"https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-compute-migrations.csv",
	"https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-operator.csv",
	"https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-network.csv",
}

var defaultKubeVirtSourceCodeTestPath = "../kubevirt/tests"

type ModelDataSources struct {
	TestDescriptor
	testFileName string
	Features     *cannier.FeatureSet
}

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
		// fetch tests from csv, attach file names, then extract feature vector
		dataSourcesByName := make(map[string]ModelDataSources)
		for _, url := range perTestExecutionCSVLinks {
			log.WithField("url", url).Infof("fetching test data")
			records, err := ExtractSourceRecords(url)
			if err != nil {
				return err
			}
			descriptors, err := DescriptorsFromRecords(records)
			if err != nil {
				return err
			}
			for _, descriptor := range descriptors {
				logTest := log.WithField("test", descriptor.Name)
				logTest.Infof("Fetching data for test %q", descriptor.Name)
				testFileName, err := ginkgo.FindTestFileByName(descriptor.Name, *testSourcePath)
				if err != nil {
					logTest.Warnf("could not find test file: %v", err)
					continue
				}
				testDescriptor, err := ginkgo.NewTestDescriptorForName(descriptor.Name, testFileName)
				if err != nil {
					logTest.Warnf("could not create descriptor for file %q: %v", testFileName, err)
					continue
				}
				logTest.Infof("extracting feature vector", descriptor.Name)
				features, err := cannier.ExtractFeatures(testDescriptor)
				if err != nil {
					return err
				}
				newDataSource := ModelDataSources{
					TestDescriptor: descriptor,
					testFileName:   testFileName,
					Features:       features,
				}
				dataSourcesByName[descriptor.Name] = newDataSource
			}
		}

		// generate model from data
		var xData [][]float64
		var yData []int
		log.Infof("generating model data")
		for _, ds := range dataSourcesByName {
			x := ds.Features.AsFloatVector()
			y := ds.TestDescriptor.Label
			xData = append(xData, x)
			yData = append(yData, int(y))
		}
		forest := randomforest.Forest{}
		forest.Data = randomforest.ForestData{X: xData, Class: yData}
		log.Infof("training model")
		forest.Train(1000)

		log.Infof("TODO: storing model")
		// 4.) store model to disk via serialization
		// https://stackoverflow.com/questions/51300011/how-to-write-a-struct-as-binary-data-to-a-file-in-golang
		return nil
	},
}

var (
	testSourcePath *string
	outputPath     *string
)

func init() {
	generateCmd.AddCommand(modelCmd)

	testSourcePath = modelCmd.Flags().StringP("test-source-path", "t", defaultKubeVirtSourceCodeTestPath, "Help message for toggle")
	outputPath = modelCmd.Flags().StringP("output-path", "o", "/tmp/kubevirt-cannier.bin", "Help message for toggle")
}

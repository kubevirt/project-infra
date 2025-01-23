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
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"kubevirt.io/project-infra/robots/pkg/cannier"
	"kubevirt.io/project-infra/robots/pkg/ginkgo"
	"time"
)

var perTestExecutionCSVLinks = []string{
	"https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-compute.csv",
	"https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-storage.csv",
	"https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-compute-migrations.csv",
	"https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-operator.csv",
	"https://storage.googleapis.com/kubevirt-prow/reports/per-test-results/kubevirt/kubevirt/last-six-months/periodic-kubevirt-e2e-k8s-1.31-sig-network.csv",
}

var (
	testSourcePath       *string
	outputBinaryFilepath *string
	onlyTestSubset       *bool
)

// generateModelCmd represents the generate command
var generateModelCmd = &cobra.Command{
	Use:   "model",
	Short: "generate model generates the random forest model from a set of test cases",
	Long: `generate model generates the random forest model from a set of test cases

It determines categories from the set of data available via per-test-execution, 
then uses the test names to extract the feature vectors, and then creates the 
dataset for the model. It finally trains the model and stores the model as a 
serialized blob to given path.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		modelLog := log.WithField("cmd", "model")

		modelData, err := GenerateModelData(modelLog)
		if err != nil {
			return err
		}

		startTime := time.Now()
		logElapsedTime := func(e *log.Entry) *log.Entry {
			return e.WithField("elapsed", fmt.Sprintf("%v", time.Now().Sub(startTime)))
		}

		logElapsedTime(modelLog).Infof("boruta")
		importantFeatures, mapOfFeatures := modelData.Boruta()
		log.Infof("important features:\n\t%v, map of features:\n\t%v", importantFeatures, mapOfFeatures)

		forest := modelData.Model()

		logElapsedTime(modelLog).Infof("training model")
		forest.Train(1000)

		logElapsedTime(modelLog).Infof("storing model data")
		err = modelData.Store(*outputBinaryFilepath)
		if err != nil {
			return err
		}

		logElapsedTime(modelLog).Infof("loading model data")
		storedDataModel, err := Load(*outputBinaryFilepath)
		if err != nil {
			return err
		}

		model := storedDataModel.Model()
		model.Train(1000)

		return nil
	},
}

func init() {
	generateCmd.AddCommand(generateModelCmd)

	testSourcePath = generateModelCmd.Flags().StringP("test-source-path", "t", "../kubevirt/tests", "Help message for toggle")
	outputBinaryFilepath = generateModelCmd.Flags().StringP("output-binary-filepath", "o", "/tmp/kubevirt-cannier-model-data.yaml", "output path for binary file to write model data to")
	onlyTestSubset = generateModelCmd.Flags().Bool("test-subset", false, "Help message for toggle")
}

func GenerateModelData(modelLog *log.Entry) (*ModelData, error) {
	modelLog.Infof("fetching test data")

	// fetch tests from csv, attach file names, then extract feature vector
	dataSourcesByName := make(map[string]TestDataPool)
	for i, url := range perTestExecutionCSVLinks {
		if *onlyTestSubset && i > 0 {
			break
		}
		urlLog := modelLog.WithField("url", url)
		urlLog.Infof("fetching test data")
		records, err := ExtractSourceRecords(url)
		if err != nil {
			return nil, err
		}
		descriptors, err := DescriptorsFromRecords(records)
		if err != nil {
			return nil, err
		}
		for k, descriptor := range descriptors {
			if *onlyTestSubset && k > 20 {
				break
			}
			logTest := urlLog.WithField("test", descriptor.Name)
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
			logTest.Infof("extracting feature vector for test %q", descriptor.Name)
			features, err := cannier.ExtractFeatures(testDescriptor)
			if err != nil {
				return nil, err
			}
			dataSourcesByName[descriptor.Name] = TestDataPool{
				TestDescriptor: descriptor,
				testFileName:   testFileName,
				Features:       features,
			}
		}
	}

	// generate model from data
	modelData := &ModelData{}
	log.Infof("generating model data")
	for _, ds := range dataSourcesByName {
		modelData.Append(ds.Features.AsFloatVector(), int(ds.TestDescriptor.Label))
	}
	return modelData, nil
}

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
	"encoding/json"
	"fmt"
	"kubevirt.io/project-infra/robots/pkg/cannier"
	"kubevirt.io/project-infra/robots/pkg/ginkgo"
	"os"

	"github.com/spf13/cobra"
)

var (
	testName            *string
	fileName            *string
	outputFileName      *string
	overwriteOutputFile *bool
	asRequest           *bool
)

func init() {
	extractCmd.AddCommand(extractFeatureSetCmd)
	testName = extractFeatureSetCmd.Flags().StringP("test-name", "t", "", "name of the test to analyze")
	fileName = extractFeatureSetCmd.Flags().StringP("filename", "f", "", "filename for the test to analyze")
	outputFileName = extractFeatureSetCmd.Flags().StringP("output-filename", "o", "", "filename to write the resulting feature set into, format is json")
	overwriteOutputFile = extractFeatureSetCmd.Flags().BoolP("overwrite", "F", false, "whether to overwrite the output file if it exists")
	asRequest = extractFeatureSetCmd.Flags().BoolP("as-request", "r", true, "whether to output the bare data or the data suitable for a hosted model request")
}

// extractFeatureSetCmd represents the extract command
var extractFeatureSetCmd = &cobra.Command{
	Use:   "features",
	Short: "features extracts a feature set from a single test",
	Long:  `Extracts a feature set as described in the CANNIER paper from a single test.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return ExtractFeatures(*testName, *fileName, *outputFileName, *overwriteOutputFile, *asRequest)
	},
}

func ExtractFeatures(testName string, fileName string, outputFileName string, overwriteOutputFile bool, asRequest bool) error {
	testDescriptor, err := ginkgo.NewTestDescriptorForName(testName, fileName)
	if err != nil {
		return err
	}
	if outputFileName == "" {
		return fmt.Errorf("output fileName is required")
	}
	_, err = os.Stat(outputFileName)
	if !overwriteOutputFile && (err == nil || !os.IsNotExist(err)) {
		return fmt.Errorf("output file %q must not exist", outputFileName)
	}
	features, err := cannier.ExtractFeatures(testDescriptor)
	if err != nil {
		return fmt.Errorf("error extracting features: %w", err)
	}
	outputFile, err := os.Create(outputFileName)
	if err != nil {
		return fmt.Errorf("error writing output file: %w", err)
	}
	defer outputFile.Close()
	if asRequest {
		err = json.NewEncoder(outputFile).Encode(RequestData{Features: features})
	} else {
		err = json.NewEncoder(outputFile).Encode(features)
	}
	if err != nil {
		return fmt.Errorf("error writing output file: %w", err)
	}
	return nil
}

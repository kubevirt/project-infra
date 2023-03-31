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
 * Copyright 2023 Red Hat, Inc.
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io/fs"
	test_label_analyzer "kubevirt.io/project-infra/robots/pkg/test-label-analyzer"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// statsCmd represents the stats command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Generates stats over test categories",
	Long:  `TODO`,
	RunE: func(_ *cobra.Command, _ []string) error {
		return runStatsCommand(rootConfigOpts)
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}

func runStatsCommand(configurationOptions configOptions) error {
	err := configurationOptions.validate()
	if err != nil {
		return err
	}

	if len(configurationOptions.ginkgoOutlinePathes) > 0 {
		jsonOutput, err := collectStatsFromGinkgoOutlines(configurationOptions)
		if err != nil {
			return err
		}
		fmt.Printf(jsonOutput)
		return nil
	}

	if configurationOptions.testFilePath != "" {

		var testOutlines []*test_label_analyzer.GinkgoNode
		err := filepath.Walk(configurationOptions.testFilePath, func(path string, info fs.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			ginkgoCommand := exec.Command("ginkgo", "outline", "--format", "json", path)
			output, err := ginkgoCommand.Output()
			if err != nil {
				e := err.(*exec.ExitError)
				stdErr := string(e.Stderr)
				if strings.Contains(stdErr, "file does not import \"github.com/onsi/ginkgo/v2\"") {
					return nil
				}
				return fmt.Errorf("command %v failed on %s: %v\n%v", ginkgoCommand, path, err, stdErr)
			}
			testOutline, err := toOutline(output)
			if err != nil {
				return fmt.Errorf("toOutline failed on %s: %v", path, err)
			}
			testOutlines = append(testOutlines, testOutline...)
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk test file path %q: %v", configurationOptions.testFilePath, err)
		}

		testStats := test_label_analyzer.GetStatsFromGinkgoOutline(configNamesToConfigs[configurationOptions.configName], testOutlines)
		marshal, err := json.Marshal(testStats)
		if err != nil {
			return err
		}
		fmt.Printf(string(marshal))
		return nil
	}

	return fmt.Errorf("not implemented")
}

func collectStatsFromGinkgoOutlines(configurationOptions configOptions) (string, error) {

	// collect the test outline data from the files and merge it into one slice
	var testOutlines []*test_label_analyzer.GinkgoNode
	for _, path := range configurationOptions.ginkgoOutlinePathes {
		fileData, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read file %q: %v", path, err)
		}
		testOutline, err2 := toOutline(fileData)
		if err2 != nil {
			return "", fmt.Errorf("failed to unmarshal file %q: %v", path, err)
		}
		testOutlines = append(testOutlines, testOutline...)
	}

	testStats := test_label_analyzer.GetStatsFromGinkgoOutline(configNamesToConfigs[configurationOptions.configName], testOutlines)
	marshal, err := json.Marshal(testStats)
	if err != nil {
		return "", err
	}

	jsonOutput := string(marshal)
	return jsonOutput, nil
}

func toOutline(fileData []byte) (testOutline []*test_label_analyzer.GinkgoNode, err error) {
	err = json.Unmarshal(fileData, &testOutline)
	return testOutline, err
}

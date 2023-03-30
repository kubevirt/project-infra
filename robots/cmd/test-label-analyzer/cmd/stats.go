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
	test_label_analyzer "kubevirt.io/project-infra/robots/pkg/test-label-analyzer"
	"os"
)

// statsCmd represents the stats command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Generates stats over test categories",
	Long:  `TODO`,
	RunE:  runStatsCommand,
}

func init() {
	rootCmd.AddCommand(statsCmd)
}

func runStatsCommand(_ *cobra.Command, _ []string) error {
	err := configOpts.validate()
	if err != nil {
		return err
	}

	// collect the test outline data from the files and merge it into one slice
	var testOutlines []*test_label_analyzer.GinkgoNode
	for _, filepath := range configOpts.ginkgoOutlinePathes {
		fileData, err := os.ReadFile(filepath)
		if err != nil {
			return fmt.Errorf("failed to read file %q: %v", filepath, err)
		}
		var testOutline []*test_label_analyzer.GinkgoNode
		err = json.Unmarshal(fileData, &testOutline)
		if err != nil {
			return fmt.Errorf("failed to unmarshal file %q: %v", filepath, err)
		}
		testOutlines = append(testOutlines, testOutline...)
	}

	testStats := test_label_analyzer.GetStatsFromGinkgoOutline(configNamesToConfigs[configOpts.configName], testOutlines)
	marshal, err := json.Marshal(testStats)
	if err != nil {
		return err
	}

	fmt.Printf(string(marshal))

	return nil
}

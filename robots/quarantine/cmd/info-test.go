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

	"github.com/spf13/cobra"
)

const (
	shortInfoTest = `Retrieves information for a test in a codebase given the test name`
)

var infoTestCmd = &cobra.Command{
	Use:   "info",
	Short: shortInfoTest,
	Long: shortInfoTest + `.

Provides available information about the hierarchy of gingko test nodes leading
to the leaf test node, like location of the nodes including file names and line
numbers, labels, types etc.

All the information provided is derived using the ginkgo dry-run mechanism.`,
	RunE: InfoTest,
}

func init() {
	infoTestCmd.PersistentFlags().StringVar(&quarantineOpts.testName, "test-name", "", "the name of the test to retrieve information for")
}

func InfoTest(_ *cobra.Command, _ []string) error {

	matchingSpecReport, err := getDryRunSpecReport(&quarantineOpts)
	if err != nil {
		return fmt.Errorf("failed to get test spec report: %w", err)
	}

	bytes, err := json.Marshal(matchingSpecReport)
	if err != nil {
		return fmt.Errorf("failed to marshal test spec report: %w", err)
	}
	fmt.Print(string(bytes))

	return nil
}

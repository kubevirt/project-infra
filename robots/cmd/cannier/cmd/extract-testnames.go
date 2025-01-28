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

import "github.com/spf13/cobra"

func init() {
	extractCmd.AddCommand(extractTestNamesCmd)
}

var extractTestNamesCmd = &cobra.Command{
	Use:   "testnames",
	Short: "testnames extracts the names for the changed ginkgo tests for a range of commits",
	Long: `Extracts the names for the changed ginkgo tests for a range of commits.

Test names are determined by looking at the changes from the lines changed in the commits, then matching those with the ginkgo outline for the changed files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return ExtractTestNames()
	},
}

func ExtractTestNames() error {
	return nil
}

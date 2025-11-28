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

package filter

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	"kubevirt.io/project-infra/pkg/git"
	testlabelanalyzer "kubevirt.io/project-infra/pkg/test-label-analyzer"
)

const FILEMODE_UGO_READ_WRITE = 0666

type filterStatsMatchingTestsOptions struct {
	outputFilePath string

	outputFile   *os.File
	inputVersion string
}

func (o *filterStatsMatchingTestsOptions) validate() error {
	if o.outputFilePath == "" {
		return fmt.Errorf("output-file needs to be provided")
	}
	_, err := os.Stat(o.outputFilePath)
	if !os.IsNotExist(err) {
		return fmt.Errorf("output file %q exists", o.outputFilePath)
	}
	o.outputFile, err = os.Create(o.outputFilePath)
	if err != nil {
		return fmt.Errorf("could not create output file %q: %w", o.outputFilePath, err)
	}
	return nil
}

var filterStatsMatchingTestsOpts = &filterStatsMatchingTestsOptions{}

var statsMatchingTestsCmd = &cobra.Command{
	Use:   "matching-tests",
	Short: "Filters the output of the stats command for matching tests only",
	Long: `Filters the output of the stats command for matching tests only.

Takes as input the first argument, parses it and converts it into a condensed output that only holds basic test
information for matching tests, like the name as it would appear inside a Junit XML file or inside a Prow Job run
overview.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return fmt.Errorf("no input file provided as argument")
		}
		err := filterStatsMatchingTestsOpts.validate()
		if err != nil {
			return fmt.Errorf("failed to validate flags: %w", err)
		}
		return filterMatchingTestsFromInputFile(args[0], filterStatsMatchingTestsOpts)
	},
}

func filterMatchingTestsFromInputFile(inputFile string, opts *filterStatsMatchingTestsOptions) error {
	fileContent, err := os.ReadFile(inputFile)
	if errors.IsNotFound(err) {
		return fmt.Errorf("input file %q does not exist", inputFile)
	} else if err != nil {
		return fmt.Errorf("input file %q created an error: %w", inputFile, err)
	}
	var input testlabelanalyzer.TestFilesStats
	err = json.Unmarshal(fileContent, &input)
	if err != nil {
		return fmt.Errorf("input file %q created an error: %w", inputFile, err)
	}
	filtered := filterMatchingTests(input, opts.inputVersion)
	marshal, err := json.Marshal(filtered)
	if err != nil {
		return fmt.Errorf("failed marshalling data: %w", err)
	}
	err = os.WriteFile(opts.outputFilePath, marshal, FILEMODE_UGO_READ_WRITE)
	return err
}

type matchingTests []matchingTest

type matchingTest struct {

	// Id has the regular expression that matched the test name
	Id string `json:"id"`

	// Reason has the reason from the rule file that explain why this test is quarantined
	Reason string `json:"reason"`

	// Version refers to the version this test is quarantined in
	Version string `json:"version,omitempty"`

	// TestName is the target test name that is matched by the regexp
	TestName string `json:"test_name"`

	// BlameLine contains the git information about the rule
	*git.BlameLine `json:"git_blame_line"`
}

func filterMatchingTests(input testlabelanalyzer.TestFilesStats, inputVersion string) matchingTests {
	var result matchingTests
	for _, fileStats := range input.FilesStats {
		for _, matchingSpecPath := range fileStats.MatchingSpecPaths {
			var name string
			for _, path := range matchingSpecPath.Path {
				name = strings.Trim(fmt.Sprintf("%s %s", name, path.Text), " ")
			}
			result = append(result, matchingTest{
				Id:        matchingSpecPath.MatchingCategory.TestNameLabelRE.String(),
				Reason:    matchingSpecPath.MatchingCategory.Name,
				Version:   inputVersion,
				TestName:  name,
				BlameLine: matchingSpecPath.MatchingCategory.BlameLine,
			})
		}
	}
	return result
}

func init() {
	statsCmd.AddCommand(statsMatchingTestsCmd)
	statsMatchingTestsCmd.PersistentFlags().StringVar(&filterStatsMatchingTestsOpts.outputFilePath, "output-file", "", "the output file to write the filtered test data to")
	statsMatchingTestsCmd.PersistentFlags().StringVar(&filterStatsMatchingTestsOpts.inputVersion, "input-version", "", "the version to place in the output file, i.e. `4.11`")
}

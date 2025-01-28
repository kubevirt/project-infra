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
	"kubevirt.io/project-infra/robots/pkg/ginkgo"
	"kubevirt.io/project-infra/robots/pkg/git"
	"os"
	"path/filepath"
	"regexp"
)

var revisionRange *string
var repoPath *string
var testSubDirectory *string

var revisionRangeRegex = regexp.MustCompile(`^([^\s]+)(..([^\s]+))?$`)

func init() {
	extractCmd.AddCommand(extractTestNamesCmd)
	revisionRange = extractTestNamesCmd.Flags().StringP("revision-range", "r", "main..HEAD", "gives the revision range to look at when determining the changes")
	repoPath = extractTestNamesCmd.Flags().StringP("repo-path", "p", "", "gives the test directory to look at when determining the changed tests")
	testSubDirectory = extractTestNamesCmd.Flags().StringP("test-subdirectory", "t", "", "gives the test directory to look at when determining the changed tests")
}

var extractTestNamesCmd = &cobra.Command{
	Use:   "testnames",
	Short: "Extracts the names for the changed ginkgo tests for a range of commits",
	Long: `Extracts the names for the changed ginkgo tests for a range of commits.

Test names are determined by looking at the changes from the lines changed in the commits, then matching those with the ginkgo outline for the changed files.`,
	RunE: func(cmd *cobra.Command, args []string) error {

		return ExtractTestNames(*revisionRange, *testSubDirectory, *repoPath)
	},
}

func ExtractTestNames(revisionRange string, testDirectory string, repoPath string) error {
	if !revisionRangeRegex.MatchString(revisionRange) {
		return fmt.Errorf("revision range must be a valid git revision range")
	}
	commits, err := git.LogCommits(revisionRange, repoPath, testDirectory)
	if err != nil {
		return err
	}
	outlines := make(map[string][]*ginkgo.Node)
	blameLines := make(map[string][]*git.BlameLine)
	for _, logCommit := range commits {
		for _, fileChange := range logCommit.FileChanges {
			testFilename := filepath.Join(repoPath, fileChange.Filename)
			_, ok := outlines[testFilename]
			if ok {
				continue
			}
			outline, err := ginkgo.OutlineFromFile(testFilename)
			if err != nil {
				return err
			}
			outlines[testFilename] = outline
			blameLinesForFile, err := git.GetBlameLinesForFile(testFilename)
			if err != nil {
				return err
			}
			blameLines[testFilename] = blameLinesForFile
		}
	}
	fmt.Printf("outlines:")
	encoder := json.NewEncoder(os.Stdout)
	encoder.Encode(&outlines)
	fmt.Printf("blameLines:")
	encoder.Encode(&blameLines)
	return nil
}

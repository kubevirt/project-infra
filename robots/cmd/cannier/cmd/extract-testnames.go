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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"kubevirt.io/project-infra/robots/pkg/ginkgo"
	"kubevirt.io/project-infra/robots/pkg/git"
	"os"
	"path/filepath"
	"regexp"
)

// flag variables
var (
	revisionRange    *string
	repoPath         *string
	testSubDirectory *string
	debug            *bool
)

var revisionRangeRegex = regexp.MustCompile(`^([^\s]+)(..([^\s]+))?$`)

func init() {
	extractCmd.AddCommand(extractTestNamesCmd)
	revisionRange = extractTestNamesCmd.Flags().StringP("revision-range", "r", "main..HEAD", "gives the revision range to look at when determining the changes")
	repoPath = extractTestNamesCmd.Flags().StringP("repo-path", "p", "", "gives the test directory to look at when determining the changed tests")
	testSubDirectory = extractTestNamesCmd.Flags().StringP("test-subdirectory", "t", "", "gives the test directory to look at when determining the changed tests")
	debug = extractTestNamesCmd.Flags().BoolP("debug", "D", false, "print and store debugging information - WARNING: might be VERY verbose!")
}

var extractTestNamesCmd = &cobra.Command{
	Use:   "testnames",
	Short: "Extracts the names for the changed ginkgo tests for a range of commits",
	Long: `Extracts the names for the changed ginkgo tests for a range of commits.

Test names are determined by looking at the changes from the lines changed in the commits, then matching those with the ginkgo outline for the changed files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.SetFormatter(&log.JSONFormatter{})
		if *debug {
			log.SetLevel(log.DebugLevel)
		}
		return ExtractTestNames(*revisionRange, *testSubDirectory, *repoPath, *debug)
	},
}

func ExtractTestNames(revisionRange string, testDirectory string, repoPath string, debug bool) error {
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
	if debug {
		commitsTemp, err := os.CreateTemp("", "commits-*.json")
		if err != nil {
			return err
		}
		defer commitsTemp.Close()
		json.NewEncoder(commitsTemp).Encode(&commits)
		log.Debugf("commits written to %q", commitsTemp.Name())
		outlinesTemp, err := os.CreateTemp("", "outlines-*.json")
		if err != nil {
			return err
		}
		defer outlinesTemp.Close()
		json.NewEncoder(outlinesTemp).Encode(&outlines)
		log.Debugf("outlines written to %q", outlinesTemp.Name())
		blameLinesTemp, err := os.CreateTemp("", "blame-lines-*.json")
		if err != nil {
			return err
		}
		json.NewEncoder(blameLinesTemp).Encode(&blameLines)
		log.Debugf("blameLines written to %q", blameLinesTemp.Name())
	}
	return nil
}

func extractChangedTestNames(commits []*git.LogCommit,
	outlines map[string][]*ginkgo.Node, blameLines map[string][]*git.BlameLine) []string {
	return nil
}

func blameLinesForCommits(commits []*git.LogCommit, blameLines map[string][]*git.BlameLine) (filenamesToBlamelines map[string][]*git.BlameLine) {
	filenamesToBlamelines = make(map[string][]*git.BlameLine)
	commitIDs := make(map[string]struct{})
	for _, commit := range commits {
		commitIDs[commit.Hash[:11]] = struct{}{}
	}

	for filename, blameLinesForFile := range blameLines {
		for _, line := range blameLinesForFile {
			if _, ok := commitIDs[line.CommitID]; !ok {
				continue
			}
			filenamesToBlamelines[filename] = append(filenamesToBlamelines[filename], line)
		}
	}

	return
}

func outlinesForBlameLines(blamelines map[string][]*git.BlameLine, outlines map[string][]*ginkgo.Node) (result []*ginkgo.Node) {
	for blameFilename, _ := range blamelines {
		if _, ok := outlines[blameFilename]; !ok {
			continue
		}

		// match the outline to the blameLine
		// problem: blameLine has a lineNo, where outline has characterNo start and end
		// therefore make a list of all lines with character start

		// as result return the filtered outline, meaning an outline with all containers
		// affected by the changes, this way the caller can construct the full test names
		// directly

	}
	return
}

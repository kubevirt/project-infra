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
 * Copyright The KubeVirt Authors.
 *
 */

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/onsi/ginkgo/v2/types"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"kubevirt.io/project-infra/pkg/ginkgo"
	"kubevirt.io/project-infra/pkg/git"
)

// flag variables
var (
	revisionRange          *string
	repoPath               *string
	testSubDirectory       *string
	outputPathForTestNames *string
	outputPathForTestPaths *string
	debug                  *bool
)

var revisionRangeRegex = regexp.MustCompile(`^([^\s]+)(..([^\s]+))?$`)

func init() {
	rootCmd.AddCommand(extractChangedTestsCmd)
	revisionRange = extractChangedTestsCmd.Flags().StringP("revision-range", "r", "main..HEAD", "revision range")
	repoPath = extractChangedTestsCmd.Flags().StringP("repo-path", "p", "", "path to the repository")
	outputPathForTestNames = extractChangedTestsCmd.Flags().StringP("output-names-path", "o", "", "path to the output json file containing the test names")
	outputPathForTestPaths = extractChangedTestsCmd.Flags().StringP("output-paths-path", "O", "", "path to the output json file containing the ginkgo node data")
	testSubDirectory = extractChangedTestsCmd.Flags().StringP("test-subdirectory", "t", "", "sub-directory of repo-path containing the tests")
	debug = extractChangedTestsCmd.Flags().BoolP("debug", "D", false, "print and store debugging information")
}

var extractChangedTestsCmd = &cobra.Command{
	Use:   "changed",
	Short: "Extracts the changed ginkgo tests for a range of commits",
	Long: `Extracts the changed ginkgo tests for a range of commits.

Changed tests are determined by looking at updated lines changed in the commits and
matching those with the ginkgo outline for the changed files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.SetFormatter(&log.JSONFormatter{})
		if *debug {
			log.SetLevel(log.DebugLevel)
		}
		return extractChangedTests(*debug, *revisionRange, *testSubDirectory, *repoPath, *outputPathForTestNames, *outputPathForTestPaths)
	},
}

func extractChangedTests(debug bool, revisionRange string, testDirectory string, repoPath string, outputTestNamesPath string, outputTestPathsPath string) error {
	if !revisionRangeRegex.MatchString(revisionRange) {
		return fmt.Errorf("revision range must be a valid git revision range")
	}
	commits, err := git.LogCommits(revisionRange, repoPath, testDirectory)
	if err != nil {
		return err
	}
	outlines := make(map[string][]*ginkgo.Node)
	blameLines := make(map[string][]*git.BlameLine)
	testfileContents := make(map[string]string)
	for _, logCommit := range commits {
		for _, fileChange := range logCommit.FileChanges {
			if !strings.HasSuffix(fileChange.Filename, ".go") {
				continue
			}
			_, ok := outlines[fileChange.Filename]
			if ok {
				continue
			}
			testfileFullPath := filepath.Join(repoPath, fileChange.Filename)
			switch fileChange.ChangeType {
			case git.Deleted:
				outlines[fileChange.Filename] = []*ginkgo.Node{}
				testfileContents[fileChange.Filename] = ""
				blameLines[fileChange.Filename] = []*git.BlameLine{}
			default:
				testfileContent, err := os.ReadFile(testfileFullPath)
				if err != nil {
					return err
				}
				outline, err := ginkgo.OutlineFromFile(testfileFullPath)
				if err != nil {
					return err
				}
				if len(outline) == 0 {
					continue
				}
				outlines[fileChange.Filename] = outline
				testfileContents[fileChange.Filename] = string(testfileContent)
				blameLinesForFile, err := git.GetBlameLinesForFile(testfileFullPath)
				if err != nil {
					return err
				}
				blameLines[fileChange.Filename] = blameLinesForFile
			}
		}
	}
	if debug {
		commitsTemp, err := os.CreateTemp("", "commits-*.json")
		if err != nil {
			return err
		}
		defer commitsTemp.Close()
		err = json.NewEncoder(commitsTemp).Encode(&commits)
		if err != nil {
			return err
		}
		log.Debugf("commits written to %q", commitsTemp.Name())
		outlinesTemp, err := os.CreateTemp("", "outlines-*.json")
		if err != nil {
			return err
		}
		defer outlinesTemp.Close()
		err = json.NewEncoder(outlinesTemp).Encode(&outlines)
		if err != nil {
			return err
		}
		log.Debugf("outlines written to %q", outlinesTemp.Name())
		blameLinesTemp, err := os.CreateTemp("", "blame-lines-*.json")
		if err != nil {
			return err
		}
		err = json.NewEncoder(blameLinesTemp).Encode(&blameLines)
		if err != nil {
			return err
		}
		log.Debugf("blameLines written to %q", blameLinesTemp.Name())
		testfileContentsTemp, err := os.CreateTemp("", "testfile-contents-*.json")
		if err != nil {
			return err
		}
		err = json.NewEncoder(testfileContentsTemp).Encode(&testfileContents)
		if err != nil {
			return err
		}
		log.Debugf("testfile contents written to %q", testfileContentsTemp.Name())
	}
	allPaths := extractChangedTestPaths(commits, outlines, blameLines, testfileContents)
	outputTestNamesFile, err := createFile(outputTestNamesPath, "changed-tests-*.json")
	if err != nil {
		return err
	}
	defer outputTestNamesFile.Close()
	changedTestNames, err := generateTestNames(allPaths, filepath.Join(repoPath, testDirectory))
	if err != nil {
		return err
	}
	err = json.NewEncoder(outputTestNamesFile).Encode(&changedTestNames)
	if err != nil {
		return err
	}
	log.Infof("test name output written to %q", outputTestNamesFile.Name())
	if err != nil {
		return err
	}
	outputTestPathsFile, err := createFile(outputTestPathsPath, "changed-test-paths-*.json")
	if err != nil {
		return err
	}
	defer outputTestPathsFile.Close()
	err = json.NewEncoder(outputTestPathsFile).Encode(&allPaths)
	if err != nil {
		return err
	}
	log.Infof("test path output written to %q", outputTestPathsFile.Name())
	return nil
}

func createFile(outputPath string, pattern string) (file *os.File, err error) {
	if outputPath == "" {
		file, err = os.CreateTemp("", pattern)
	} else {
		file, err = os.Create(outputPath)
	}
	return
}

func generateTestNames(allPaths [][]*ginkgo.Node, testFilePath string) ([]string, error) {
	reports, _, err := ginkgo.DryRun(testFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not run ginkgo dry-run in folder %q: %w", testFilePath, err)
	}
	if reports == nil {
		return nil, fmt.Errorf("no results for ginkgo dry-run in folder %q: %w", testFilePath, err)
	}

	var allNodeTexts [][]string
	for _, path := range allPaths {
		var nodeTexts []string
		for _, node := range path {
			if node.Text == "undefined" {
				continue
			}
			nodeTexts = append(nodeTexts, node.Text)
		}
		allNodeTexts = append(allNodeTexts, nodeTexts)
	}

	matchingSpecReports := filterMatchingSpecsByPartsContainingTestNames(reports, allNodeTexts)

	var testnames []string
	for _, r := range matchingSpecReports {
		var texts []string
		texts = append(texts, r.ContainerHierarchyTexts...)
		texts = append(texts, r.LeafNodeText)
		testname := strings.Join(texts, " ")
		testnames = append(testnames, testname)
	}
	return testnames, nil
}

func filterMatchingSpecsByPartsContainingTestNames(reports []types.Report, allNodeTexts [][]string) []types.SpecReport {
	matchingSpecReports := ginkgo.FilterSpecReports(reports, func(r types.SpecReport) bool {
		var specReportTexts []string
		specReportTexts = append(specReportTexts, r.ContainerHierarchyTexts...)
		specReportTexts = append(specReportTexts, r.LeafNodeText)
		if len(specReportTexts) == 1 && specReportTexts[0] == "" {
			return false
		}
		for _, nodeTexts := range allNodeTexts {
			nodeTextsIndex := len(nodeTexts) - 1
			containsAll := true
			for specIndex := len(specReportTexts) - 1; specIndex >= 0 && nodeTextsIndex >= 0; specIndex-- {
				if !strings.Contains(specReportTexts[specIndex], nodeTexts[nodeTextsIndex]) {
					containsAll = false
					break
				}
				nodeTextsIndex--
			}
			if containsAll {
				return true
			}
		}
		return false
	}, -1)
	return matchingSpecReports
}

type CommitHashMap interface {
	HasCommitID(hash string) bool
}

type commitHashMapImpl struct {
	hashes []string
}

func (c commitHashMapImpl) HasCommitID(hash string) bool {
	for _, commit := range c.hashes {
		if strings.HasPrefix(commit, hash) {
			return true
		}
	}
	return false
}

func NewHashMap(hashes []string) CommitHashMap {
	return &commitHashMapImpl{
		hashes: hashes,
	}
}

func extractChangedTestPaths(commits []*git.LogCommit, outlines map[string][]*ginkgo.Node, blameLines map[string][]*git.BlameLine, testfileContents map[string]string) [][]*ginkgo.Node {
	filenames := map[string]struct{}{}
	var commitHashes []string
	for _, commit := range commits {
		commitHashes = append(commitHashes, commit.Hash)
		for _, change := range commit.FileChanges {
			filenames[change.Filename] = struct{}{}
		}
	}
	hashMap := NewHashMap(commitHashes)

	var allPaths [][]*ginkgo.Node
	for filename := range filenames {
		if outlinesForFilename, ok := outlines[filename]; !ok || len(outlinesForFilename) == 0 {
			continue
		}
		var lines []int
		for _, blame := range blameLines[filename] {
			if !hashMap.HasCommitID(blame.CommitID) {
				continue
			}
			lines = append(lines, blame.LineNo)
		}
		mapper := OutlineMapper{
			lineModel: newLineModel(testfileContents[filename]),
			outline:   outlines[filename],
		}
		pathsForLines, err := mapper.GetPathsForLines(lines...)
		if err != nil {
			log.Fatalf("fatal error: %v", err)
		}
		if len(pathsForLines) == 0 {
			continue
		}
		allPaths = append(allPaths, pathsForLines...)
	}
	return allPaths
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

type CharRange struct {
	Start, End int
}

type LineModel struct {
	lines []string
	start []int
	end   []int
}

func newLineModel(content string) (m *LineModel) {
	m = &LineModel{}
	m.lines = strings.Split(content, "\n")
	charIndex := 0
	for _, line := range m.lines {
		m.start = append(m.start, charIndex)
		m.end = append(m.end, charIndex+len(line))
		charIndex += len(line) + 1
	}
	return
}

func (m *LineModel) GetCharRangeForLine(lineNo int) *CharRange {
	return &CharRange{
		Start: m.start[lineNo-1],
		End:   m.end[lineNo-1],
	}
}

func generateLineModelFromFile(testFilepath string) (*LineModel, error) {
	wholeFileContent, err := os.ReadFile(testFilepath)
	if err != nil {
		return nil, err
	}
	return newLineModel(string(wholeFileContent)), nil
}

type OutlineMapper struct {
	lineModel *LineModel
	outline   []*ginkgo.Node
}

func (m *OutlineMapper) GetPathsForLines(lines ...int) ([][]*ginkgo.Node, error) {
	sort.Ints(lines)

	var charRanges []*CharRange
	for _, line := range lines {
		charRanges = append(charRanges, m.lineModel.GetCharRangeForLine(line))
	}

	outlineMatchingCharRanges := findMatchingChildren(charRanges, m.outline)
	paths := expandPaths(nil, outlineMatchingCharRanges)

	return paths, nil
}

func expandPaths(parents []*ginkgo.Node, children []*ginkgo.Node) (paths [][]*ginkgo.Node) {
	for _, child := range children {
		if len(child.Nodes) > 0 {
			newParents := append(parents, child.CloneWithoutNodes())
			nodes := expandPaths(newParents, child.Nodes)
			paths = append(paths, nodes...)
		} else {
			paths = append(paths, append(parents, child))
		}
	}
	return
}

func findMatchingChildren(charRanges []*CharRange, nodes []*ginkgo.Node) []*ginkgo.Node {
	var matchingNodes []*ginkgo.Node
	for _, node := range nodes {
		foundMatchingCharRange := false
		for _, charRange := range charRanges {
			if node.Start > charRange.End || node.End <= charRange.End {
				continue
			}
			foundMatchingCharRange = true
			break
		}
		if !foundMatchingCharRange {
			continue
		}
		matchingChildren := findMatchingChildren(charRanges, node.Nodes)
		if matchingChildren == nil {
			matchingNodes = append(matchingNodes, node.CloneWithNodes(func(n *ginkgo.Node) bool { return n.Name != "By" }))
		} else {
			clone := node.CloneWithoutNodes()
			clone.Nodes = matchingChildren
			matchingNodes = append(matchingNodes, clone)
		}
	}
	return matchingNodes
}

func generateOutlineMapperFromFiles(testFilepath string, outline []*ginkgo.Node) (m *OutlineMapper, err error) {
	var lineModel *LineModel
	lineModel, err = generateLineModelFromFile(testFilepath)
	if err != nil {
		return
	}
	m = &OutlineMapper{lineModel: lineModel, outline: outline}

	return
}

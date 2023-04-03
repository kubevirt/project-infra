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
	"path"
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

		testFileOutlines := map[string][]*test_label_analyzer.GinkgoNode{}
		err := filepath.Walk(configurationOptions.testFilePath, func(path string, info fs.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			testOutline, err2 := getGinkgoOutlineFromFile(path)
			if err2 != nil {
				return err2
			}
			if testOutline == nil {
				return nil
			}
			testFileOutlines[path] = testOutline
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk test file path %q: %v", configurationOptions.testFilePath, err)
		}

		config, err := configurationOptions.getConfig()
		if err != nil {
			return err
		}

		var testFilesStats []*test_label_analyzer.FileStats
		for testFilePath, testFileOutline := range testFileOutlines {
			testStatsForFile := test_label_analyzer.GetStatsFromGinkgoOutline(config, testFileOutline)
			file, err := os.ReadFile(testFilePath)
			if err != nil {
				// Should only happen if the file has been deleted after the outline has been retrieved
				panic(err)
			}
			testFileContent := string(file)
			for _, matchingSpecPathes := range testStatsForFile.MatchingSpecPathes {

				var lineNos []int
				offset := 0
				lineNo := 1
				for _, node := range matchingSpecPathes.Path {
					lineNo += newlineCount(testFileContent, offset, node.Start)
					lineNos = append(lineNos, lineNo)
					offset = node.Start + 1
				}
				matchingSpecPathes.Lines = lineNos

				blameArgs := []string{"blame", filepath.Base(testFilePath)}
				for _, blameLineNo := range lineNos {
					blameArgs = append(blameArgs, fmt.Sprintf("-L %d,%d", blameLineNo, blameLineNo))
				}
				command := exec.Command("git", blameArgs...)
				command.Dir = filepath.Dir(testFilePath)
				output, err := command.Output()
				if err != nil {
					e := err.(*exec.ExitError)
					return fmt.Errorf("exec %v failed: %v", command, e)
				}
				matchingSpecPathes.GitBlameLines = test_label_analyzer.ExtractGitBlameInfo(strings.Split(string(output), "\n"))
			}
			testFilesStats = append(testFilesStats, &test_label_analyzer.FileStats{
				RemoteURL: path.Join(configurationOptions.remoteURL, strings.TrimPrefix(strings.TrimPrefix(testFilePath, configurationOptions.testFilePath), "/")),
				Config:    config,
				TestStats: testStatsForFile,
			})
		}
		marshal, err := json.Marshal(testFilesStats)
		if err != nil {
			return err
		}
		fmt.Printf(string(marshal))
		return nil
	}

	return fmt.Errorf("not implemented")
}

func newlineCount(s string, start int, end int) int {
	n := 0
	for i := start; i < len(s) && i < end; i++ {
		if s[i] == '\n' {
			n++
		}
	}
	return n
}

func getGinkgoOutlineFromFile(path string) ([]*test_label_analyzer.GinkgoNode, error) {
	ginkgoCommand := exec.Command("ginkgo", "outline", "--format", "json", path)
	output, err := ginkgoCommand.Output()
	if err != nil {
		e := err.(*exec.ExitError)
		stdErr := string(e.Stderr)
		if strings.Contains(stdErr, "file does not import \"github.com/onsi/ginkgo/v2\"") {
			return nil, nil
		}
		return nil, fmt.Errorf("command %v failed on %s: %v\n%v", ginkgoCommand, path, err, stdErr)
	}
	testOutline, err := toOutline(output)
	if err != nil {
		return nil, fmt.Errorf("toOutline failed on %s: %v", path, err)
	}
	return testOutline, nil
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

	config, err := configurationOptions.getConfig()
	if err != nil {
		return "", err
	}
	testStats := test_label_analyzer.GetStatsFromGinkgoOutline(config, testOutlines)
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

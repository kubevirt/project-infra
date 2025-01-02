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
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/onsi/ginkgo/v2/ginkgo/outline"
	"github.com/spf13/cobra"
	"html/template"
	"io"
	"io/fs"
	"kubevirt.io/project-infra/robots/pkg/git"
	testlabelanalyzer "kubevirt.io/project-infra/robots/pkg/test-label-analyzer"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const shortStatsDescription = "Generates stats over test categories"

// statsCmd represents the stats command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: shortStatsDescription,
	Long: shortStatsDescription + `

Can either emit json or html format data about the targeted tests.
`,
	RunE: func(_ *cobra.Command, _ []string) error {
		return runStatsCommand(rootConfigOpts)
	},
}

//go:embed stats.gohtml
var statsHTMLTemplate string

func init() {
	RootCmd.AddCommand(statsCmd)
}

type TestHTMLData struct {
	*testlabelanalyzer.Config `json:"config"`

	// MatchingPath holds the test_label_analyzer.PathStats that matched the node
	MatchingPath *testlabelanalyzer.PathStats

	// RemoteURL is the absolute path to the file, most certainly an absolute URL inside a version control repository
	// containing a commit ID in order to exactly define the state of the file that was traversed
	RemoteURL string `json:"path"`

	// ElementsMatchingConfig contains whether each of the GitBlameLines matches the *test_label_analyzer.Config
	ElementsMatchingConfig []bool

	// Permalinks contain the links that point to the commits related to the GitBlameLines
	Permalinks []string

	// Age contain the textual representations of time passed since the change was done in GitBlameLines
	Age []string
}

// initElementsMatchingConfig initializes ElementsMatchingConfig with the indices of the GitBlameLines that are
// matching the *test_label_analyzer.Config
func (t *TestHTMLData) initElementsMatchingConfig() {
	if len(t.ElementsMatchingConfig) > 0 {
		panic("t.ElementsMatchingConfig already initialized")
	}
	t.ElementsMatchingConfig = make([]bool, len(t.MatchingPath.GitBlameLines))
	wholeLine := ""
	for index, node := range t.MatchingPath.Path {
		wholeLine = strings.TrimSpace(wholeLine + " " + node.Text)
		matchString := t.MatchingPath.MatchingCategory.TestNameLabelRE.MatchString(wholeLine)
		if matchString {
			t.ElementsMatchingConfig[index] = true
		}
	}
}

var replaceToPermaLinkMatcher = regexp.MustCompile("https://github.com/[\\w]+/[\\w]+/((tree|blob)/[\\w]+)/.*")

// initPermalinks initializes Permalinks with the permanent links to the GitBlameLines
func (t *TestHTMLData) initPermalinks() {
	if len(t.Permalinks) > 0 {
		panic("t.Permalinks already initialized")
	}
	submatch := replaceToPermaLinkMatcher.FindStringSubmatch(t.RemoteURL)
	if len(submatch) < 2 {
		return
	}
	for _, gitLine := range t.MatchingPath.GitBlameLines {
		permaLink := strings.ReplaceAll(t.RemoteURL, submatch[1], fmt.Sprintf("commit/%s", gitLine.CommitID))
		t.Permalinks = append(t.Permalinks, permaLink)
	}
}

// initAge initializes Age with the textual description of the GitBlameLines age
func (t *TestHTMLData) initAge() {
	if len(t.Age) > 0 {
		panic("t.Age already initialized")
	}
	for _, gitLine := range t.MatchingPath.GitBlameLines {
		date := gitLine.Date
		if t.MatchingPath.MatchingCategory.BlameLine != nil {
			date = t.MatchingPath.MatchingCategory.BlameLine.Date
		}
		t.Age = append(t.Age, testlabelanalyzer.Since(date))
	}
}

func (t *TestHTMLData) collectEarliestChangeDateFromGitLines() time.Time {
	if len(t.ElementsMatchingConfig) == 0 {
		panic("t.ElementsMatchingConfig not initialized")
	}
	changeDate := time.Now()
	for index, matches := range t.ElementsMatchingConfig {
		if !matches {
			continue
		}
		if len(t.MatchingPath.GitBlameLines) <= index {
			continue
		}
		if t.MatchingPath.GitBlameLines[index].Date.Before(changeDate) {
			changeDate = t.MatchingPath.GitBlameLines[index].Date
		}
	}
	return changeDate
}

type StatsHTMLData struct {
	TestHTMLData         []*TestHTMLData
	Date                 time.Time
	ConfigurationOptions ConfigOptions
}

func (s *StatsHTMLData) Len() int {
	return len(s.TestHTMLData)
}

func (s *StatsHTMLData) Less(i, k int) bool {
	if s.TestHTMLData[i].MatchingPath.MatchingCategory.BlameLine != nil && s.TestHTMLData[k].MatchingPath.MatchingCategory.BlameLine != nil {
		return s.TestHTMLData[i].MatchingPath.MatchingCategory.BlameLine.Date.Before(s.TestHTMLData[k].MatchingPath.MatchingCategory.BlameLine.Date)
	}
	return s.TestHTMLData[i].collectEarliestChangeDateFromGitLines().Before(s.TestHTMLData[k].collectEarliestChangeDateFromGitLines())
}

func (s *StatsHTMLData) Swap(i, k int) {
	s.TestHTMLData[i], s.TestHTMLData[k] = s.TestHTMLData[k], s.TestHTMLData[i]
}

func NewStatsHTMLData(stats []*testlabelanalyzer.FileStats, configurationOptions ConfigOptions) *StatsHTMLData {
	statsHTMLData := &StatsHTMLData{
		Date:                 time.Now(),
		ConfigurationOptions: configurationOptions,
	}
	for _, fileStats := range stats {
		for _, path := range fileStats.TestStats.MatchingSpecPaths {
			statsHTMLData.TestHTMLData = append(statsHTMLData.TestHTMLData, newTestHTMLData(fileStats, path))
		}
	}
	sort.Sort(statsHTMLData)
	return statsHTMLData
}

func newTestHTMLData(fileStats *testlabelanalyzer.FileStats, path *testlabelanalyzer.PathStats) *TestHTMLData {
	testHTMLData := &TestHTMLData{
		MatchingPath: path,
		RemoteURL:    fileStats.RemoteURL,
	}
	testHTMLData.initElementsMatchingConfig()
	testHTMLData.initPermalinks()
	testHTMLData.initAge()
	return testHTMLData
}

func runStatsCommand(configurationOptions ConfigOptions) error {
	err := configurationOptions.validate()
	if err != nil {
		return err
	}

	if len(configurationOptions.ginkgoOutlinePaths) > 0 {
		jsonOutput, err := collectStatsFromGinkgoOutlines(configurationOptions)
		if err != nil {
			return err
		}
		fmt.Printf(jsonOutput)
		return nil
	}

	if configurationOptions.testFilePath != "" {

		testFileOutlines, err := getTestFileOutlines(configurationOptions)
		if err != nil {
			return fmt.Errorf("failed to walk test file path %q: %w", configurationOptions.testFilePath, err)
		}
		if len(testFileOutlines) == 0 {
			return fmt.Errorf("could not derive an outline, tests are likely not Ginkgo V2 based")
		}

		config, err := configurationOptions.getConfig()
		if err != nil {
			return err
		}

		filesStats, err := generateStatsFromOutlinesWithGitBlameInfo(configurationOptions, testFileOutlines, config)
		if err != nil {
			return err
		}

		if !configurationOptions.outputHTML {
			testFilesStats := testlabelanalyzer.TestFilesStats{
				FilesStats: filesStats,
				Config:     config,
			}
			data, err := json.Marshal(testFilesStats)
			if err != nil {
				return err
			}
			fmt.Printf(string(data))
			return nil
		}

		htmlTemplate, err := template.New("statsWithGitBlameInfo").Parse(statsHTMLTemplate)
		if err != nil {
			return err
		}

		statsHTMLData := NewStatsHTMLData(filesStats, configurationOptions)
		err = htmlTemplate.Execute(os.Stdout, statsHTMLData)
		return err
	}

	return fmt.Errorf("not implemented")
}

func generateStatsFromOutlinesWithGitBlameInfo(configurationOptions ConfigOptions, testFileOutlines map[string][]*testlabelanalyzer.GinkgoNode, config *testlabelanalyzer.Config) ([]*testlabelanalyzer.FileStats, error) {
	var testFilesStats []*testlabelanalyzer.FileStats
	for testFilePath, testFileOutline := range testFileOutlines {
		testStatsForFile := testlabelanalyzer.GetStatsFromGinkgoOutline(config, testFileOutline)
		file, err := os.ReadFile(testFilePath)
		if err != nil {
			// Should only happen if the file has been deleted after the outline has been retrieved
			panic(err)
		}
		testFileContent := string(file)
		for _, matchingSpecPathes := range testStatsForFile.MatchingSpecPaths {

			var lineNos []int
			offset := 0
			lineNo := 1
			for _, node := range matchingSpecPathes.Path {
				lineNo += newlineCount(testFileContent, offset, node.Start)
				lineNos = append(lineNos, lineNo)
				offset = node.Start + 1
			}
			matchingSpecPathes.Lines = lineNos

			gitBlameInfo, err3 := git.GetBlameLinesForFile(testFilePath, lineNos...)
			if err3 != nil {
				return nil, err3
			}
			matchingSpecPathes.GitBlameLines = gitBlameInfo
			if len(matchingSpecPathes.GitBlameLines) == 0 {
				return nil, fmt.Errorf("git blame lines extraction failed!")
			}
		}
		testFilesStats = append(testFilesStats, &testlabelanalyzer.FileStats{
			RemoteURL: fmt.Sprintf("%s/%s", strings.TrimSuffix(configurationOptions.remoteURL, "/"), strings.TrimPrefix(strings.TrimPrefix(testFilePath, configurationOptions.testFilePath), "/")),
			TestStats: testStatsForFile,
		})
	}
	return testFilesStats, nil
}

func getTestFileOutlines(configurationOptions ConfigOptions) (map[string][]*testlabelanalyzer.GinkgoNode, error) {
	testFileOutlines := map[string][]*testlabelanalyzer.GinkgoNode{}
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
	return testFileOutlines, err
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

func getGinkgoOutlineFromFile(path string) ([]*testlabelanalyzer.GinkgoNode, error) {

	// since there's no output catchable from the command, we need to use pipe
	// and redirect the output
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	buildOutlineCommand := outline.BuildOutlineCommand()
	buildOutlineCommand.Run([]string{"--format", "json", path}, nil)

	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, err := io.Copy(&buf, r)
		if err != nil {
			panic(err)
		}
		outC <- buf.String()
	}()

	// restore the output to normal
	err := w.Close()
	os.Stdout = old
	out := <-outC
	output := []byte(out)

	if err != nil {
		switch err.(type) {
		case *exec.ExitError:
			e := err.(*exec.ExitError)
			stdErr := string(e.Stderr)
			if strings.Contains(stdErr, "file does not import \"github.com/onsi/ginkgo/v2\"") {
				return nil, nil
			}
			return nil, fmt.Errorf("exec %v failed: %s", buildOutlineCommand, e.Stderr)
		case *exec.Error:
			e := err.(*exec.Error)
			return nil, fmt.Errorf(`exec "%v" failed: %s`, buildOutlineCommand, e)
		default:
			return nil, fmt.Errorf(`exec "%v" failed: %s`, buildOutlineCommand, err)
		}
	}
	testOutline, err := toOutline(output)
	if err != nil {
		return nil, fmt.Errorf("toOutline failed on %s: %w", path, err)
	}
	return testOutline, nil
}

func collectStatsFromGinkgoOutlines(configurationOptions ConfigOptions) (string, error) {

	// collect the test outline data from the files and merge it into one slice
	var testOutlines []*testlabelanalyzer.GinkgoNode
	for _, path := range configurationOptions.ginkgoOutlinePaths {
		fileData, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("failed to read file %q: %w", path, err)
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
	testStats := testlabelanalyzer.GetStatsFromGinkgoOutline(config, testOutlines)
	marshal, err := json.Marshal(testStats)
	if err != nil {
		return "", err
	}

	jsonOutput := string(marshal)
	return jsonOutput, nil
}

func toOutline(fileData []byte) (testOutline []*testlabelanalyzer.GinkgoNode, err error) {
	err = json.Unmarshal(fileData, &testOutline)
	return testOutline, err
}

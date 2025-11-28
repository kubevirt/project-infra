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
	osexec "os/exec"
	"path/filepath"
	"slices"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	ginkgo2 "kubevirt.io/project-infra/pkg/ginkgo"
	git2 "kubevirt.io/project-infra/pkg/git"
)

type TestDataContainer struct {
	commits          []*git2.LogCommit
	outlines         map[string][]*ginkgo2.Node
	blameLines       map[string][]*git2.BlameLine
	testfileContents map[string]string
}

func (c *TestDataContainer) deserialize() {
	commitsFile, err := os.OpenFile("testdata/mapping/commits.json", os.O_RDONLY, 0666)
	Expect(err).ToNot(HaveOccurred())
	err = json.NewDecoder(commitsFile).Decode(&c.commits)
	Expect(err).ToNot(HaveOccurred())
	outlinesFile, err := os.OpenFile("testdata/mapping/outlines.json", os.O_RDONLY, 0666)
	Expect(err).ToNot(HaveOccurred())
	err = json.NewDecoder(outlinesFile).Decode(&c.outlines)
	Expect(err).ToNot(HaveOccurred())
	blameLinesFile, err := os.OpenFile("testdata/mapping/blame-lines.json", os.O_RDONLY, 0666)
	Expect(err).ToNot(HaveOccurred())
	err = json.NewDecoder(blameLinesFile).Decode(&c.blameLines)
	Expect(err).ToNot(HaveOccurred())
	testfileContentsFile, err := os.OpenFile("testdata/mapping/testfile-contents.json", os.O_RDONLY, 0666)
	Expect(err).ToNot(HaveOccurred())
	err = json.NewDecoder(testfileContentsFile).Decode(&c.testfileContents)
	Expect(err).ToNot(HaveOccurred())
	Expect(c.commits).ToNot(BeNil())
	Expect(c.outlines).ToNot(BeNil())
	Expect(c.blameLines).ToNot(BeNil())
	Expect(c.testfileContents).ToNot(BeNil())
}

func (c *TestDataContainer) data() (commits []*git2.LogCommit, outlines map[string][]*ginkgo2.Node, blameLines map[string][]*git2.BlameLine, testfileContents map[string]string) {
	return c.commits, c.outlines, c.blameLines, c.testfileContents
}

var _ = Describe("extract testnames", func() {

	When("asking for blame lines", Ordered, func() {

		var filenamesToBlamelines map[string][]*git2.BlameLine

		BeforeAll(func() {
			container := &TestDataContainer{}
			container.deserialize()
			commits, _, blameLines, _ := container.data()
			filenamesToBlamelines = blameLinesForCommits(commits, blameLines)
		})

		It("gets a result when asking for blame lines for a set of commits", func() {
			Expect(filenamesToBlamelines).ToNot(BeEmpty())
		})

		It("gets a result when asking for blame lines for a set of commits", func() {
			Expect(filenamesToBlamelines["tests/guestlog/guestlog.go"]).ToNot(BeEmpty())
		})

	})

	When("generating a line model from test file", Ordered, func() {

		var lineModel *LineModel

		BeforeAll(func() {
			var err error
			lineModel, err = generateLineModelFromFile("testdata/simple_test.go")
			Expect(err).ToNot(HaveOccurred())
		})

		It("generates model", func() {
			Expect(lineModel).ToNot(BeNil())
		})

		It("fetches char range for line", func() {
			Expect(lineModel.GetCharRangeForLine(31)).ToNot(BeNil())
		})

		It("fetches char range for line", func() {
			line := lineModel.GetCharRangeForLine(31)
			Expect(line.Start).To(BeEquivalentTo(837))
			Expect(line.End).To(BeEquivalentTo(882))
		})

	})

	When("creating a mapper for a line change to an outline path", Ordered, func() {

		var outlineMapper *OutlineMapper

		BeforeAll(func() {
			var err error
			var outline []*ginkgo2.Node

			file, err := os.Open("testdata/simple_test_outline.json")
			Expect(err).ToNot(HaveOccurred())
			err = json.NewDecoder(file).Decode(&outline)
			Expect(err).ToNot(HaveOccurred())

			outlineMapper, err = generateOutlineMapperFromFiles("testdata/simple_test.go", outline)
			Expect(err).ToNot(HaveOccurred())

		})

		It("is created", func() {
			Expect(outlineMapper).ToNot(BeNil())
		})

		It("returns an outline path", func() {
			Expect(outlineMapper.GetPathsForLines(31)).ToNot(BeNil())
		})

		It("outline path has one node for line", func() {
			Expect(outlineMapper.GetPathsForLines(31)).To(HaveLen(1))
		})

		It("outline path has two nodes for two line", func() {
			Expect(outlineMapper.GetPathsForLines(31, 39)).To(HaveLen(2))
		})

		It("outline path has two nodes for line inside parent node", func() {
			Expect(outlineMapper.GetPathsForLines(53)).To(HaveLen(2))
		})

	})

	When("extracting (real) test names", Ordered, func() {

		var container *TestDataContainer

		BeforeAll(func() {
			container = &TestDataContainer{}
			container.deserialize()
		})

		It("should have a result", func() {
			Expect(extractChangedTestPaths(container.data())).ToNot(HaveLen(0))
		})

		It("should have exactly one result", func() {
			Expect(extractChangedTestPaths(container.data())).To(HaveLen(1))
		})

	})

	When("extracting test names from generated outline", Ordered, func() {

		const (
			testfileName        = "simple_test.go"
			testfilePartialPath = "testdata"
			testfilePath        = "robots/cmd/ginkgo-tests/cmd/testdata"
		)

		var (
			outlines         map[string][]*ginkgo2.Node
			blameLines       map[string][]*git2.BlameLine
			testfileContents map[string]string
			testPaths        [][]*ginkgo2.Node
			changedTestNames []string
		)

		BeforeAll(func() {

			absBasePath, err := filepath.Abs("./../../../..")
			Expect(err).ToNot(HaveOccurred())
			absTestPath := filepath.Join(absBasePath, testfilePath)
			absTestFile := filepath.Join(absTestPath, testfileName)
			relTestFile := filepath.Join(testfilePath, testfileName)

			// determine latest commit id on test file
			// note: required to make test robust against rebases
			cmd := osexec.Command("git", "log", "--format=%H", "-1", "--", testfilePath)
			cmd.Dir = absBasePath
			output, err := cmd.Output()
			Expect(err).ToNot(HaveOccurred())

			commitID := string(output)
			commitID = strings.TrimSuffix(commitID, "\n")
			Expect(commitID).ToNot(BeEquivalentTo(""))

			commits, err := git2.LogCommits(fmt.Sprintf("%s^1..%s", commitID, commitID), absBasePath, testfilePath)
			Expect(err).ToNot(HaveOccurred())

			testOutline, err := ginkgo2.OutlineFromFile(absTestFile)
			Expect(err).ToNot(HaveOccurred())
			outlines = map[string][]*ginkgo2.Node{
				relTestFile: testOutline,
			}

			blameLinesForFile, err := git2.GetBlameLinesForFile(absTestFile)
			Expect(err).ToNot(HaveOccurred())
			blameLines = map[string][]*git2.BlameLine{
				relTestFile: blameLinesForFile,
			}

			testfileContent, err := os.ReadFile(absTestFile)
			Expect(err).ToNot(HaveOccurred())
			testfileContents = map[string]string{
				relTestFile: string(testfileContent),
			}

			testPaths = extractChangedTestPaths(commits, outlines, blameLines, testfileContents)
			changedTestNames, err = generateTestNames(testPaths, absTestPath)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should contain paths", func() {
			Expect(testPaths).ToNot(BeEmpty())
		})

		It("should contain test names", func() {
			Expect(changedTestNames).ToNot(BeEmpty())
		})

		It("no test name should contain test names", func() {
			for _, name := range changedTestNames {
				Expect(name).ToNot(ContainSubstring("undefined"))
			}
		})

		It("should contain final test name", func() {
			Expect(slices.Contains(changedTestNames, "whatever description of describe description of it")).To(BeTrue())
		})

	})

	DescribeTable("filtering matching specs",
		func(reports []types.Report, allNodeTexts [][]string, matchingSpecReports []types.SpecReport) {
			Expect(filterMatchingSpecsByPartsContainingTestNames(reports, allNodeTexts)).To(BeEquivalentTo(matchingSpecReports))
		},
		Entry(
			"empty specreport",
			[]types.Report{
				{
					SpecReports: types.SpecReports{
						{
							ContainerHierarchyTexts: []string{},
							LeafNodeText:            "",
						},
					},
				},
			},
			[][]string{},
			nil,
		),
		Entry(
			"empty empty",
			[]types.Report{},
			[][]string{},
			nil,
		),
		Entry(
			"full node match",
			[]types.Report{
				{
					SpecReports: types.SpecReports{
						{
							ContainerHierarchyTexts: []string{
								"Describe",
								"When",
							},
							LeafNodeText: "[test_id:42]",
						},
					},
				},
			},
			[][]string{
				{
					"Describe",
					"When",
					"[test_id:42]",
				},
			},
			[]types.SpecReport{
				{
					ContainerHierarchyTexts: []string{
						"Describe",
						"When",
					},
					LeafNodeText: "[test_id:42]",
				},
			},
		),
		Entry(
			"leaf node only",
			[]types.Report{
				{
					SpecReports: types.SpecReports{
						{
							ContainerHierarchyTexts: []string{
								"Describe",
								"When",
							},
							LeafNodeText: "[test_id:42]",
						},
					},
				},
			},
			[][]string{
				{"[test_id:42]"},
			},
			[]types.SpecReport{
				{
					ContainerHierarchyTexts: []string{
						"Describe",
						"When",
					},
					LeafNodeText: "[test_id:42]",
				},
			},
		),
		Entry(
			"node matches subset",
			[]types.Report{
				{
					SpecReports: types.SpecReports{
						{
							ContainerHierarchyTexts: []string{
								"whatever the main topic is",
								"whenever we do something",
							},
							LeafNodeText: "[test_id:42] this is the full description",
						},
					},
				},
			},
			[][]string{
				{
					"main topic",
					"do something",
					"full description",
				},
			},
			[]types.SpecReport{
				{
					ContainerHierarchyTexts: []string{
						"whatever the main topic is",
						"whenever we do something",
					},
					LeafNodeText: "[test_id:42] this is the full description",
				},
			},
		),
		Entry(
			"node doesn't match subset",
			[]types.Report{
				{
					SpecReports: types.SpecReports{
						{
							ContainerHierarchyTexts: []string{
								"whatever the main topic is",
								"whenever we do something",
							},
							LeafNodeText: "[test_id:42] this is the full description",
						},
					},
				},
			},
			[][]string{
				{
					"another topic",
					"do something",
					"full description",
				},
			},
			nil,
		),
	)

})

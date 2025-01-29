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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/project-infra/robots/pkg/ginkgo"
	"kubevirt.io/project-infra/robots/pkg/git"
	"os"
)

type TestDataContainer struct {
	commits          []*git.LogCommit
	outlines         map[string][]*ginkgo.Node
	blameLines       map[string][]*git.BlameLine
	testfileContents map[string]string
}

func (c *TestDataContainer) deserialize() {
	commitsFile, err := os.OpenFile("testdata/mapping/commits.json", os.O_RDONLY, 666)
	Expect(err).ToNot(HaveOccurred())
	err = json.NewDecoder(commitsFile).Decode(&c.commits)
	Expect(err).ToNot(HaveOccurred())
	outlinesFile, err := os.OpenFile("testdata/mapping/outlines.json", os.O_RDONLY, 666)
	Expect(err).ToNot(HaveOccurred())
	err = json.NewDecoder(outlinesFile).Decode(&c.outlines)
	Expect(err).ToNot(HaveOccurred())
	blameLinesFile, err := os.OpenFile("testdata/mapping/blame-lines.json", os.O_RDONLY, 666)
	Expect(err).ToNot(HaveOccurred())
	err = json.NewDecoder(blameLinesFile).Decode(&c.blameLines)
	Expect(err).ToNot(HaveOccurred())
	testfileContentsFile, err := os.OpenFile("testdata/mapping/testfile-contents.json", os.O_RDONLY, 666)
	Expect(err).ToNot(HaveOccurred())
	err = json.NewDecoder(testfileContentsFile).Decode(&c.testfileContents)
	Expect(err).ToNot(HaveOccurred())
	Expect(c.commits).ToNot(BeNil())
	Expect(c.outlines).ToNot(BeNil())
	Expect(c.blameLines).ToNot(BeNil())
	Expect(c.testfileContents).ToNot(BeNil())
}

func (c *TestDataContainer) data() (commits []*git.LogCommit, outlines map[string][]*ginkgo.Node, blameLines map[string][]*git.BlameLine, testfileContents map[string]string) {
	return c.commits, c.outlines, c.blameLines, c.testfileContents
}

var _ = Describe("extract testnames", func() {

	When("asking for blame lines", Ordered, func() {

		var filenamesToBlamelines map[string][]*git.BlameLine

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
			var outline []*ginkgo.Node

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

		It("should find the full test name that was changed", func() {
			expectedPartialTestName := "it should not skip any log line even trying to flood the serial console for QOSGuaranteed VMs"
			paths := extractChangedTestPaths(container.data())
			testNames := generateTestNames(paths)
			Expect(testNames[0]).To(ContainSubstring(expectedPartialTestName))
		})

	})

})

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
	commits    []*git.LogCommit
	outlines   map[string][]*ginkgo.Node
	blameLines map[string][]*git.BlameLine
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
	Expect(c.commits).ToNot(BeNil())
	Expect(c.outlines).ToNot(BeNil())
	Expect(c.blameLines).ToNot(BeNil())
}

func (c *TestDataContainer) data() (commits []*git.LogCommit, outlines map[string][]*ginkgo.Node, blameLines map[string][]*git.BlameLine) {
	return c.commits, c.outlines, c.blameLines
}

var _ = Describe("extract testnames", func() {

	When("asking for blame lines", Ordered, func() {

		var filenamesToBlamelines map[string][]*git.BlameLine

		BeforeAll(func() {
			container := &TestDataContainer{}
			container.deserialize()
			commits, _, blameLines := container.data()
			filenamesToBlamelines = blameLinesForCommits(commits, blameLines)
		})

		It("gets a result when asking for blame lines for a set of commits", func() {
			Expect(filenamesToBlamelines).ToNot(BeEmpty())
		})

		It("gets a result when asking for blame lines for a set of commits", func() {
			Expect(filenamesToBlamelines["tests/guestlog/guestlog.go"]).ToNot(BeEmpty())
		})

	})

	FWhen("matching blame lines to outlines", Ordered, func() {

		var allOutlinesForBlameLines []*ginkgo.Node

		BeforeAll(func() {
			container := &TestDataContainer{}
			container.deserialize()
			commits, outlines, blameLines := container.data()
			filenamesToBlamelines := blameLinesForCommits(commits, blameLines)
			allOutlinesForBlameLines = outlinesForBlameLines(filenamesToBlamelines, outlines)
		})

		It("returns a non empty result", func() {
			Expect(allOutlinesForBlameLines).ToNot(BeNil())
		})

	})

	PWhen("extracting test names", Ordered, func() {

		var container *TestDataContainer

		BeforeAll(func() {
			container = &TestDataContainer{}
			container.deserialize()
		})

		It("should find the full test name that was changed", func() {
			expectedPartialTestName := "it should not skip any log line even trying to flood the serial console for QOSGuaranteed VMs"
			Expect(extractChangedTestNames(container.data())).To(ContainSubstring(expectedPartialTestName))
		})

	})

})

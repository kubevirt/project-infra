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

package ginkgo

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("dry-run", func() {
	Context("from directory", func() {
		It("returns report", func() {
			report, output, err := DryRun("testdata")
			Expect(err).ToNot(HaveOccurred())
			Expect(report).ToNot(BeEmpty())
			Expect(output).ToNot(BeNil())
		})
	})
})

var _ = Describe("extract", func() {
	Context("labels", func() {
		DescribeTable("returns a set of labels",
			func(r SpecReport, expected []string, m []LabelMatcher) {
				Expect(ExtractLabels(r, m...)).To(BeEquivalentTo(expected))
			},
			Entry("leaf node label",
				SpecReport{
					LeafNodeLabels: []string{
						"sig-compute",
					},
				},
				[]string{"sig-compute"},
				nil,
			),
			Entry("parent label",
				SpecReport{
					ContainerHierarchyLabels: [][]string{{"sig-compute"}},
				},
				[]string{"sig-compute"},
				nil,
			),
			Entry("all labels",
				SpecReport{
					ContainerHierarchyLabels: [][]string{{"sig-compute"}},
					LeafNodeLabels:           []string{"sig-compute"},
				},
				[]string{"sig-compute", "sig-compute"},
				nil,
			),
			Entry("matching labels",
				SpecReport{
					ContainerHierarchyLabels: [][]string{{"sig-compute"}},
					LeafNodeLabels:           []string{"whatever"},
				},
				[]string{"sig-compute"},
				[]LabelMatcher{NewRegexLabelMatcher("sig-.*")},
			),
			Entry("matching labels where leaf node contains match",
				SpecReport{
					ContainerHierarchyLabels: [][]string{{"whatever"}},
					LeafNodeLabels:           []string{"sig-compute"},
				},
				[]string{"sig-compute"},
				[]LabelMatcher{NewRegexLabelMatcher("sig-.*")},
			),
		)
	})
})

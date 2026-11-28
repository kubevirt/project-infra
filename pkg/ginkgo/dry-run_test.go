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

var _ = Describe("quarantined by name", func() {
	const testName = "[sig-compute] Infrastructure tls configuration should enforce TLS configuration on virt-template components"

	DescribeTable("it",
		func(r SpecReport, expected bool) {
			Expect(QuarantinedByName(testName)(r)).To(Equal(expected))
		},
		Entry("matches a test with a quarantined leaf node",
			SpecReport{
				ContainerHierarchyTexts: []string{"[sig-compute] Infrastructure", "tls configuration"},
				LeafNodeText:            "[QUARANTINE]should enforce TLS configuration on virt-template components",
			},
			true,
		),
		Entry("matches a test with a quarantined container node",
			SpecReport{
				ContainerHierarchyTexts: []string{"[sig-compute] Infrastructure", "[QUARANTINE]tls configuration"},
				LeafNodeText:            "should enforce TLS configuration on virt-template components",
			},
			true,
		),
		Entry("does not match a test that is not quarantined",
			SpecReport{
				ContainerHierarchyTexts: []string{"[sig-compute] Infrastructure", "tls configuration"},
				LeafNodeText:            "should enforce TLS configuration on virt-template components",
			},
			false,
		),
		Entry("does not match a different quarantined test",
			SpecReport{
				ContainerHierarchyTexts: []string{"[sig-compute] Infrastructure", "tls configuration"},
				LeafNodeText:            "[QUARANTINE]should enforce TLS configuration on virt-operator",
			},
			false,
		),
	)

	It("matches a test with the quarantine label in the middle of a node text", func() {
		r := SpecReport{
			ContainerHierarchyTexts: []string{"[sig-compute][USB] [QUARANTINE] host USB Passthrough"},
			LeafNodeText:            "should start a VMI with a USB device",
		}
		Expect(QuarantinedByName("[sig-compute][USB] host USB Passthrough should start a VMI with a USB device")(r)).To(BeTrue())
	})

	It("does not modify the node texts of the spec report", func() {
		r := SpecReport{
			ContainerHierarchyTexts: []string{"[sig-compute] Infrastructure", "[QUARANTINE]tls configuration"},
			LeafNodeText:            "should enforce TLS configuration on virt-template components",
		}
		Expect(QuarantinedByName(testName)(r)).To(BeTrue())
		Expect(r.ContainerHierarchyTexts).To(Equal([]string{"[sig-compute] Infrastructure", "[QUARANTINE]tls configuration"}))
	})

	It("returns the quarantined spec report", func() {
		reports := []Report{
			{
				SpecReports: []SpecReport{
					{
						ContainerHierarchyTexts: []string{"[sig-compute] Infrastructure", "tls configuration"},
						LeafNodeText:            "should enforce TLS configuration on virt-operator",
					},
					{
						ContainerHierarchyTexts: []string{"[sig-compute] Infrastructure", "tls configuration"},
						LeafNodeText:            "[QUARANTINE]should enforce TLS configuration on virt-template components",
					},
				},
			},
		}
		Expect(GetSpecReportByTestName(reports, testName)).To(BeNil())
		r := GetQuarantinedSpecReportByTestName(reports, testName)
		Expect(r).ToNot(BeNil())
		Expect(r.LeafNodeText).To(Equal("[QUARANTINE]should enforce TLS configuration on virt-template components"))
	})
})

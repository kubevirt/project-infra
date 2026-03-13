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
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	flakestats "kubevirt.io/project-infra/pkg/flake-stats"
	"kubevirt.io/project-infra/pkg/searchci"
	"os"
	"strings"
	"time"
)

var _ = Describe("auto-quarantine", func() {
	When("Writing pr description", func() {
		var tempFile *os.File
		BeforeEach(func() {
			var err error
			tempFile, err = os.CreateTemp("", "pr-description-test-*.md")
			Expect(err).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			Expect(os.Remove(tempFile.Name())).ToNot(HaveOccurred())
		})
		DescribeTable("contains expected",
			func(testsPerSIG TestsPerSIG, expectedSubstrings []string) {
				Expect(writePRDescriptionToFile(tempFile.Name(), testsPerSIG)).ToNot(HaveOccurred())
				bytes, err := os.ReadFile(tempFile.Name())
				content := string(bytes)
				Expect(err).ToNot(HaveOccurred())
				for _, expectedSubstring := range expectedSubstrings {
					Expect(strings.Contains(content, expectedSubstring)).To(BeTrue(), fmt.Sprintf(`expected
"
%s
"

to contain

%s`, content, expectedSubstring))
				}
			},
			Entry("succeeds", TestsPerSIG{}, []string{}),
			Entry("iterates over sigs",
				TestsPerSIG{
					"compute": []*TestToQuarantine{},
				},
				[]string{"/sig compute"},
			),
			Entry("iterates over tests",
				TestsPerSIG{
					"compute": []*TestToQuarantine{
						{
							Test: &flakestats.TopXTest{
								Name:                   "[sig-compute] whatever test",
								AllFailures:            nil,
								FailuresPerDay:         nil,
								FailuresPerLane:        nil,
								NoteHasBeenQuarantined: false,
							},
							TimeRange:   "24h",
							SearchCIURL: "https://search.ci.kubevirt.io",
							RelevantImpacts: []searchci.Impact{
								{
									URL:          "https://relevant-impact-url",
									Percent:      42,
									URLToDisplay: "https://relevant-impact-display-url",
									BuildURLs: []searchci.JobBuildURL{
										{
											URL:      "https://job-build-url",
											Interval: 37 * time.Minute,
										},
									},
								},
							},
							SpecReport: nil,
						},
					},
				},
				[]string{
					"/sig compute",
					"[sig-compute] whatever test",
					"search.ci.kubevirt.io",
					"https://relevant-impact-url",
					"42%",
					"https://relevant-impact-display-url",
					"https://job-build-url",
					"37",
				},
			),
		)
	})
	When("grouping tests", func() {
		DescribeTable("it",
			func(testsToQuarantine []*TestToQuarantine, expected TestsPerSIG) {
				Expect(groupTestsBySIG(testsToQuarantine)).To(BeEquivalentTo(expected))
			},
			Entry("puts each into its sig group",
				[]*TestToQuarantine{
					{
						Test: &flakestats.TopXTest{
							Name: "compute test",
						},
						SpecReport: &SpecReport{
							LeafNodeLabels: []string{
								"sig-compute",
							},
						},
					},
					{
						Test: &flakestats.TopXTest{
							Name: "network test",
						},
						SpecReport: &SpecReport{
							LeafNodeLabels: []string{
								"sig-network",
							},
						},
					},
					{
						Test: &flakestats.TopXTest{
							Name: "storage test",
						},
						SpecReport: &SpecReport{
							LeafNodeLabels: []string{
								"sig-storage",
							},
						},
					},
				},
				TestsPerSIG{
					"compute": []*TestToQuarantine{
						{
							Test: &flakestats.TopXTest{
								Name: "compute test",
							},
							SpecReport: &SpecReport{
								LeafNodeLabels: []string{
									"sig-compute",
								},
							},
						},
					},
					"network": []*TestToQuarantine{
						{
							Test: &flakestats.TopXTest{
								Name: "network test",
							},
							SpecReport: &SpecReport{
								LeafNodeLabels: []string{
									"sig-network",
								},
							},
						},
					},
					"storage": []*TestToQuarantine{
						{
							Test: &flakestats.TopXTest{
								Name: "storage test",
							},
							SpecReport: &SpecReport{
								LeafNodeLabels: []string{
									"sig-storage",
								},
							},
						},
					},
				},
			),
			Entry("puts a test with multiple sigs into one group",
				[]*TestToQuarantine{
					{
						Test: &flakestats.TopXTest{
							Name: "compute storage test",
						},
						SpecReport: &SpecReport{
							LeafNodeLabels: []string{
								"sig-compute",
								"sig-storage",
							},
						},
					},
				},
				TestsPerSIG{
					"compute": []*TestToQuarantine{
						{
							Test: &flakestats.TopXTest{
								Name: "compute storage test",
							},
							SpecReport: &SpecReport{
								LeafNodeLabels: []string{
									"sig-compute",
									"sig-storage",
								},
							},
						},
					},
				},
			),
			Entry("puts a test with multiple sigs into the topmost sig group",
				[]*TestToQuarantine{
					{
						Test: &flakestats.TopXTest{
							Name: "compute storage test",
						},
						SpecReport: &SpecReport{
							ContainerHierarchyLabels: [][]string{
								{
									"sig-compute",
								},
							},
							LeafNodeLabels: []string{
								"sig-storage",
							},
						},
					},
				},
				TestsPerSIG{
					"compute": []*TestToQuarantine{
						{
							Test: &flakestats.TopXTest{
								Name: "compute storage test",
							},
							SpecReport: &SpecReport{
								ContainerHierarchyLabels: [][]string{
									{
										"sig-compute",
									},
								},
								LeafNodeLabels: []string{
									"sig-storage",
								},
							},
						},
					},
				},
			),
		)
	})
})

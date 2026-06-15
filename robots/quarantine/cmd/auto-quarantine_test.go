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
	"os"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	flakestats "kubevirt.io/project-infra/pkg/flake-stats"
	"kubevirt.io/project-infra/pkg/searchci"
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

	When("loading required job names", func() {
		var tempFile *os.File
		BeforeEach(func() {
			var err error
			tempFile, err = os.CreateTemp("", "presubmits-*.yaml")
			Expect(err).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			Expect(os.Remove(tempFile.Name())).ToNot(HaveOccurred())
		})

		DescribeTable("it",
			func(yamlContent string, orgRepo string, expectedJobs map[string]struct{}, expectErr bool) {
				_, err := tempFile.WriteString(yamlContent)
				Expect(err).ToNot(HaveOccurred())
				Expect(tempFile.Close()).ToNot(HaveOccurred())

				result, err := loadRequiredJobNames(tempFile.Name(), orgRepo)
				if expectErr {
					Expect(err).To(HaveOccurred())
					return
				}
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(expectedJobs))
			},
			Entry("returns only required jobs",
				`presubmits:
  kubevirt/kubevirt:
  - name: pull-kubevirt-e2e-k8s-1.35-sig-compute
    always_run: true
    spec:
      containers:
      - image: test
  - name: pull-kubevirt-e2e-k8s-1.36-sig-compute
    optional: true
    always_run: true
    spec:
      containers:
      - image: test
`,
				"kubevirt/kubevirt",
				map[string]struct{}{
					"pull-kubevirt-e2e-k8s-1.35-sig-compute": {},
				},
				false,
			),
			Entry("excludes skip_report jobs",
				`presubmits:
  kubevirt/kubevirt:
  - name: pull-kubevirt-e2e-k8s-1.35-sig-compute
    always_run: true
    spec:
      containers:
      - image: test
  - name: pull-kubevirt-e2e-k8s-1.35-sig-hidden
    skip_report: true
    always_run: true
    spec:
      containers:
      - image: test
`,
				"kubevirt/kubevirt",
				map[string]struct{}{
					"pull-kubevirt-e2e-k8s-1.35-sig-compute": {},
				},
				false,
			),
			Entry("fails for unknown org/repo",
				`presubmits:
  kubevirt/kubevirt:
  - name: pull-kubevirt-e2e-k8s-1.35-sig-compute
    always_run: true
    spec:
      containers:
      - image: test
`,
				"unknown/repo",
				nil,
				true,
			),
		)
	})

	When("determining tests for quarantine", func() {
		const (
			testName      = "[sig-compute] my flaky test should do something"
			requiredLane  = "pull-kubevirt-e2e-k8s-1.35-sig-compute"
			optionalLane  = "pull-kubevirt-e2e-k8s-1.36-sig-compute"
			requiredLane2 = "pull-kubevirt-e2e-k8s-1.35-sig-network"
		)

		var originalGetQuarantineCandidate func(*flakestats.TopXTest, searchci.TimeRange) (*TestToQuarantine, error)

		BeforeEach(func() {
			originalGetQuarantineCandidate = getQuarantineCandidate
			quarantineOpts.matchingLaneRegexString = defaultMatchingLaneRegexString
			quarantineOpts.releaseLaneSuffix = ""
			quarantineOpts.maxTestsToQuarantine = 0
		})

		AfterEach(func() {
			getQuarantineCandidate = originalGetQuarantineCandidate
		})

		newImpact := func(lane string, percent float64) searchci.Impact {
			return searchci.Impact{
				URL:     "https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/" + lane,
				Percent: percent,
			}
		}

		newTopXTest := func(name string) *flakestats.TopXTest {
			return flakestats.NewTopXTest(name)
		}

		reportsContaining := func(leafNodeText string, containerTexts ...string) []Report {
			return []Report{
				{
					SpecReports: []SpecReport{
						{
							LeafNodeText:            leafNodeText,
							ContainerHierarchyTexts: containerTexts,
						},
					},
				},
			}
		}

		stubCandidate := func(impacts []searchci.Impact) {
			getQuarantineCandidate = func(topXTest *flakestats.TopXTest, _ searchci.TimeRange) (*TestToQuarantine, error) {
				return &TestToQuarantine{
					Test:            topXTest,
					RelevantImpacts: impacts,
				}, nil
			}
		}

		It("filters out impacts from optional lanes and keeps required ones", func() {
			stubCandidate([]searchci.Impact{
				newImpact(requiredLane, 10),
				newImpact(optionalLane, 15),
			})

			result, err := determineTestsForQuarantine(
				flakestats.TopXTests{newTopXTest(testName)},
				reportsContaining("should do something", "[sig-compute] my flaky test"),
				map[string]struct{}{requiredLane: {}},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].RelevantImpacts).To(HaveLen(1))
			Expect(result[0].RelevantImpacts[0].URL).To(ContainSubstring(requiredLane))
		})

		It("skips candidates where no impacts match required lanes", func() {
			stubCandidate([]searchci.Impact{
				newImpact(optionalLane, 10),
			})

			result, err := determineTestsForQuarantine(
				flakestats.TopXTests{newTopXTest(testName)},
				reportsContaining("should do something", "[sig-compute] my flaky test"),
				map[string]struct{}{requiredLane: {}},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("respects maxTestsToQuarantine ceiling", func() {
			const testName2 = "[sig-network] another flaky test should work"
			stubCandidate([]searchci.Impact{
				newImpact(requiredLane, 10),
				newImpact(requiredLane2, 8),
			})
			quarantineOpts.maxTestsToQuarantine = 1

			reports := []Report{
				{
					SpecReports: []SpecReport{
						{
							LeafNodeText:            "should do something",
							ContainerHierarchyTexts: []string{"[sig-compute] my flaky test"},
						},
						{
							LeafNodeText:            "should work",
							ContainerHierarchyTexts: []string{"[sig-network] another flaky test"},
						},
					},
				},
			}

			result, err := determineTestsForQuarantine(
				flakestats.TopXTests{newTopXTest(testName), newTopXTest(testName2)},
				reports,
				map[string]struct{}{requiredLane: {}, requiredLane2: {}},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
		})

		It("skips tests with no matching spec report", func() {
			stubCandidate([]searchci.Impact{
				newImpact(requiredLane, 10),
			})

			result, err := determineTestsForQuarantine(
				flakestats.TopXTests{newTopXTest(testName)},
				reportsContaining("completely different test"),
				map[string]struct{}{requiredLane: {}},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(BeEmpty())
		})

		It("propagates error from getQuarantineCandidate", func() {
			getQuarantineCandidate = func(_ *flakestats.TopXTest, _ searchci.TimeRange) (*TestToQuarantine, error) {
				return nil, fmt.Errorf("scrape failed")
			}

			_, err := determineTestsForQuarantine(
				flakestats.TopXTests{newTopXTest(testName)},
				reportsContaining("should do something", "[sig-compute] my flaky test"),
				map[string]struct{}{requiredLane: {}},
			)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("scrape failed"))
		})

		It("filters out impacts not matching lane regex", func() {
			nonMatchingLane := "pull-kubevirt-unit-test"
			stubCandidate([]searchci.Impact{
				newImpact(nonMatchingLane, 20),
				newImpact(requiredLane, 10),
			})

			result, err := determineTestsForQuarantine(
				flakestats.TopXTests{newTopXTest(testName)},
				reportsContaining("should do something", "[sig-compute] my flaky test"),
				map[string]struct{}{nonMatchingLane: {}, requiredLane: {}},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].RelevantImpacts).To(HaveLen(1))
			Expect(result[0].RelevantImpacts[0].URL).To(ContainSubstring(requiredLane))
		})

		It("keeps required and filters optional from mixed impacts", func() {
			stubCandidate([]searchci.Impact{
				newImpact(requiredLane, 10),
				newImpact(optionalLane, 15),
				newImpact(requiredLane2, 8),
			})

			result, err := determineTestsForQuarantine(
				flakestats.TopXTests{newTopXTest(testName)},
				reportsContaining("should do something", "[sig-compute] my flaky test"),
				map[string]struct{}{requiredLane: {}, requiredLane2: {}},
			)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveLen(1))
			Expect(result[0].RelevantImpacts).To(HaveLen(2))
		})
	})
})

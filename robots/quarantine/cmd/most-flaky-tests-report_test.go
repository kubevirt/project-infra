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
	"bytes"
	"html/template"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	flakestats "kubevirt.io/project-infra/pkg/flake-stats"
	"kubevirt.io/project-infra/pkg/searchci"
)

var _ = Describe("most-flaky-tests", func() {
	When("filtering impact", func() {
		DescribeTable("by test lane",
			func(topXTest *flakestats.TopXTest, impact []searchci.Impact, expected []searchci.Impact) {
				Expect(searchci.FilterImpactsBy(impact, matchesAnyFailureLane(topXTest))).To(BeEquivalentTo(expected))
			},
			Entry("matches main lane",
				&flakestats.TopXTest{
					FailuresPerLane: map[string]*flakestats.FailureCounter{
						"pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations": {},
					},
				},
				[]searchci.Impact{
					{
						URL: "https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations",
					},
				},
				[]searchci.Impact{
					{
						URL: "https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations",
					},
				},
			),
			Entry("matches release lane",
				&flakestats.TopXTest{
					FailuresPerLane: map[string]*flakestats.FailureCounter{
						"pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations": {},
					},
				},
				[]searchci.Impact{
					{
						URL: "https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations-1.6",
					},
				},
				[]searchci.Impact{
					{
						URL: "https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.32-sig-compute-migrations-1.6",
					},
				},
			),
		)
	})

	When("aggregating most flaky tests by SIG", func() {
		var originalGetQuarantineCandidate func(*flakestats.TopXTest, searchci.TimeRange) (*TestToQuarantine, error)
		var originalOpts quarantineOptions

		BeforeEach(func() {
			originalGetQuarantineCandidate = getQuarantineCandidate
			originalOpts = quarantineOpts
			quarantineOpts.maxFailureAge = 72 * time.Hour
			quarantineOpts.minRecentFailures = 2
			quarantineOpts.minFailureInterval = 24 * time.Hour
		})

		AfterEach(func() {
			getQuarantineCandidate = originalGetQuarantineCandidate
			quarantineOpts = originalOpts
		})

		It("marks candidate with recent spread-out failures as HasRecentFailures", func() {
			getQuarantineCandidate = func(topXTest *flakestats.TopXTest, _ searchci.TimeRange) (*TestToQuarantine, error) {
				return &TestToQuarantine{
					Test: topXTest,
					RelevantImpacts: []searchci.Impact{
						{
							URL:     "https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.35-sig-compute",
							Percent: 10,
							BuildURLs: []searchci.JobBuildURL{
								{Interval: 4 * time.Hour},
								{Interval: 36 * time.Hour},
							},
						},
					},
				}, nil
			}

			_, _, result, err := aggregateMostFlakyTestsBySIG(flakestats.TopXTests{
				flakestats.NewTopXTest("[sig-compute] my recent flaky test"),
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveKey("sig-compute"))
			tests := result["sig-compute"]["[sig-compute] my recent flaky test"]
			Expect(tests).ToNot(BeEmpty())
			Expect(tests[0].HasRecentFailures).To(BeTrue())
		})

		DescribeTable("filters candidates by failure age",
			func(ages []time.Duration, maxAge time.Duration, expectedIncluded bool) {
				quarantineOpts.maxFailureAge = maxAge
				var builds []searchci.JobBuildURL
				for _, age := range ages {
					builds = append(builds, searchci.JobBuildURL{Interval: age})
				}
				getQuarantineCandidate = func(topXTest *flakestats.TopXTest, _ searchci.TimeRange) (*TestToQuarantine, error) {
					return &TestToQuarantine{
						Test: topXTest,
						RelevantImpacts: []searchci.Impact{
							{
								URL:       "https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.35-sig-compute",
								Percent:   10,
								BuildURLs: builds,
							},
						},
					}, nil
				}

				testName := "[sig-compute] my flaky test"
				sigs, testNames, result, err := aggregateMostFlakyTestsBySIG(flakestats.TopXTests{
					flakestats.NewTopXTest(testName),
				})

				Expect(err).ToNot(HaveOccurred())
				if expectedIncluded {
					Expect(sigs).To(ConsistOf("sig-compute"))
					Expect(testNames).To(ConsistOf(testName))
					tests := result["sig-compute"][testName]
					Expect(tests).To(HaveLen(len(mostFlakyTestsTimeRanges)))
					Expect(tests[0].HasRecentFailures).To(BeFalse())
				} else {
					Expect(sigs).To(BeEmpty())
					Expect(testNames).To(BeEmpty())
					Expect(result).To(BeEmpty())
				}
			},
			Entry("excludes week-old failures", []time.Duration{168 * time.Hour, 192 * time.Hour}, 72*time.Hour, false),
			Entry("includes week-old failures with a wider age limit", []time.Duration{168 * time.Hour}, 336*time.Hour, true),
			Entry("excludes candidates without builds", []time.Duration{}, 72*time.Hour, false),
			Entry("keeps a single recent failure", []time.Duration{4 * time.Hour}, 72*time.Hour, true),
			Entry("keeps a recent burst", []time.Duration{2 * time.Hour, 3 * time.Hour}, 72*time.Hour, true),
			Entry("keeps mixed recent and stale failures", []time.Duration{168 * time.Hour, 4 * time.Hour}, 72*time.Hour, true),
			Entry("keeps a failure at the age limit", []time.Duration{72 * time.Hour}, 72*time.Hour, true),
			Entry("excludes a failure beyond the age limit", []time.Duration{72*time.Hour + time.Second}, 72*time.Hour, false),
			Entry("honors a custom age limit", []time.Duration{48 * time.Hour}, 24*time.Hour, false),
		)
	})

	When("classifying optional lanes from required job names", func() {
		var data MostFlakyTestsTemplateData

		BeforeEach(func() {
			data = NewMostFlakyTestsTemplateData(nil, nil, nil, map[string]struct{}{
				"pull-kubevirt-e2e-k8s-1.36-sig-compute": {},
				"pull-kubevirt-e2e-k8s-1.36-sig-storage": {},
			})
		})

		It("treats required presubmits as not optional", func() {
			Expect(data.IsOptionalLane("pull-kubevirt-e2e-k8s-1.36-sig-compute")).To(BeFalse())
			Expect(data.IsOptionalLane("pull-kubevirt-e2e-k8s-1.36-sig-storage")).To(BeFalse())
		})

		It("treats optional presubmits as optional", func() {
			Expect(data.IsOptionalLane("pull-kubevirt-e2e-k8s-1.37-sig-compute")).To(BeTrue())
		})

		It("tags release-branch lanes as optional because they are absent from main presubmits", func() {
			Expect(data.IsOptionalLane("pull-kubevirt-e2e-k8s-1.36-sig-compute-1.6")).To(BeTrue())
		})

		DescribeTable("keeps release and optional checkboxes independent",
			func(showRelease, showOptional, isRelease, optional, expectedHidden bool) {
				hide := (!showRelease && isRelease) || (!showOptional && optional && !isRelease)
				Expect(hide).To(Equal(expectedHidden))
			},
			Entry("default hides release lanes", false, false, true, true, true),
			Entry("default hides optional main lanes", false, false, false, true, true),
			Entry("default keeps required main lanes", false, false, false, false, false),
			Entry("show release only still shows release lanes tagged optional", true, false, true, true, false),
			Entry("show release only still hides optional main lanes", true, false, false, true, true),
			Entry("show optional only still hides release lanes", false, true, true, true, true),
			Entry("show optional only shows optional main lanes", false, true, false, true, false),
			Entry("both checked shows release lanes", true, true, true, true, false),
			Entry("both checked shows optional main lanes", true, true, false, true, false),
		)

		It("treats empty job names as optional", func() {
			Expect(data.IsOptionalLane("")).To(BeTrue())
		})
	})

	When("rendering the report template", func() {
		It("emits optional-lane checkbox and data-optional attributes", func() {
			testName := "[sig-compute] flake"
			data := NewMostFlakyTestsTemplateData(
				map[string]TestsPerSIG{
					"sig-compute": {
						testName: []*TestToQuarantine{
							{
								Test:      flakestats.NewTopXTest(testName),
								TimeRange: searchci.ThreeDays,
								RelevantImpacts: []searchci.Impact{
									{
										URL:          "https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.36-sig-compute",
										URLToDisplay: "pull-kubevirt-e2e-k8s-1.36-sig-compute",
										Percent:      2,
									},
									{
										URL:          "https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.37-sig-compute",
										URLToDisplay: "pull-kubevirt-e2e-k8s-1.37-sig-compute",
										Percent:      20,
									},
									{
										URL:          "https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.36-sig-compute-1.6",
										URLToDisplay: "pull-kubevirt-e2e-k8s-1.36-sig-compute-1.6",
										Percent:      8,
									},
								},
							},
						},
					},
				},
				[]string{"sig-compute"},
				[]string{testName},
				map[string]struct{}{
					"pull-kubevirt-e2e-k8s-1.36-sig-compute": {},
				},
			)

			tmpl, err := template.New("mostFlakyTests").Parse(mostFlakyTestsReportTemplate)
			Expect(err).ToNot(HaveOccurred())
			var buf bytes.Buffer
			Expect(tmpl.Execute(&buf, data)).To(Succeed())
			html := buf.String()
			Expect(html).To(ContainSubstring(`id="show-optional-lanes"`))
			Expect(html).To(ContainSubstring(`data-lane="pull-kubevirt-e2e-k8s-1.36-sig-compute"`))
			Expect(html).To(ContainSubstring(`data-optional="false"`))
			Expect(html).To(ContainSubstring(`data-lane="pull-kubevirt-e2e-k8s-1.37-sig-compute"`))
			Expect(html).To(ContainSubstring(`data-optional="true"`))
			Expect(html).To(ContainSubstring(`data-lane="pull-kubevirt-e2e-k8s-1.36-sig-compute-1.6"`))
			Expect(html).To(ContainSubstring(`(!showOptional && optional && !isReleaseLane)`))
		})
	})
})

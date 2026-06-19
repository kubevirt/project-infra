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

		BeforeEach(func() {
			originalGetQuarantineCandidate = getQuarantineCandidate
			quarantineOpts.maxFailureAge = 72 * time.Hour
			quarantineOpts.minRecentFailures = 2
			quarantineOpts.minFailureInterval = 24 * time.Hour
		})

		AfterEach(func() {
			getQuarantineCandidate = originalGetQuarantineCandidate
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

		It("marks candidate with only stale failures as not HasRecentFailures", func() {
			getQuarantineCandidate = func(topXTest *flakestats.TopXTest, _ searchci.TimeRange) (*TestToQuarantine, error) {
				return &TestToQuarantine{
					Test: topXTest,
					RelevantImpacts: []searchci.Impact{
						{
							URL:     "https://prow.ci.kubevirt.io/job-history/kubevirt-prow/pr-logs/directory/pull-kubevirt-e2e-k8s-1.35-sig-compute",
							Percent: 10,
							BuildURLs: []searchci.JobBuildURL{
								{Interval: 264 * time.Hour},
								{Interval: 288 * time.Hour},
							},
						},
					},
				}, nil
			}

			_, _, result, err := aggregateMostFlakyTestsBySIG(flakestats.TopXTests{
				flakestats.NewTopXTest("[sig-compute] my stale flaky test"),
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(HaveKey("sig-compute"))
			tests := result["sig-compute"]["[sig-compute] my stale flaky test"]
			Expect(tests).ToNot(BeEmpty())
			Expect(tests[0].HasRecentFailures).To(BeFalse())
		})
	})
})

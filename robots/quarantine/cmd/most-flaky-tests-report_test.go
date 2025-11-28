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
})

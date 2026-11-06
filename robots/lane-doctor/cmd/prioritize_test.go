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
)

var _ = Describe("hasLabel", func() {

	DescribeTable("matches labels",
		func(labels []string, target string, expected bool) {
			Expect(hasLabel(labels, target)).To(Equal(expected))
		},
		Entry("found", []string{"lgtm", "approved"}, "lgtm", true),
		Entry("not found", []string{"lgtm", "approved"}, "hold", false),
		Entry("empty labels", []string{}, "lgtm", false),
		Entry("nil labels", nil, "lgtm", false),
	)
})

var _ = Describe("hasAnyApprovedLabel", func() {

	DescribeTable("matches approved variants",
		func(labels []string, expected bool) {
			Expect(hasAnyApprovedLabel(labels)).To(Equal(expected))
		},
		Entry("exact approved", []string{"approved"}, true),
		Entry("approved with prefix", []string{"approved-for-merge"}, true),
		Entry("no approved", []string{"lgtm", "hold"}, false),
		Entry("empty labels", []string{}, false),
		Entry("nil labels", nil, false),
	)
})

var _ = Describe("classify", func() {

	It("assigns P1 to PRs with lgtm + approved", func() {
		prs := []StuckPR{
			{Number: 1, Labels: []string{"lgtm", "approved"}},
		}
		groups := classify(prs)
		Expect(groups).To(HaveLen(1))
		Expect(groups[0].Name).To(Equal("P1"))
		Expect(groups[0].PRNumbers).To(ConsistOf(1))
	})

	It("assigns P2 to PRs with lgtm only", func() {
		prs := []StuckPR{
			{Number: 2, Labels: []string{"lgtm"}},
		}
		groups := classify(prs)
		Expect(groups).To(HaveLen(1))
		Expect(groups[0].Name).To(Equal("P2"))
		Expect(groups[0].PRNumbers).To(ConsistOf(2))
	})

	It("assigns P2 to PRs with approved only", func() {
		prs := []StuckPR{
			{Number: 3, Labels: []string{"approved"}},
		}
		groups := classify(prs)
		Expect(groups).To(HaveLen(1))
		Expect(groups[0].Name).To(Equal("P2"))
		Expect(groups[0].PRNumbers).To(ConsistOf(3))
	})

	It("assigns P3 to PRs with no merge-readiness labels", func() {
		prs := []StuckPR{
			{Number: 4, Labels: []string{"size/M"}},
		}
		groups := classify(prs)
		Expect(groups).To(HaveLen(1))
		Expect(groups[0].Name).To(Equal("P3"))
		Expect(groups[0].PRNumbers).To(ConsistOf(4))
	})

	It("assigns P4 to PRs on hold", func() {
		prs := []StuckPR{
			{Number: 5, Labels: []string{"lgtm", "approved", "do-not-merge/hold"}},
		}
		groups := classify(prs)
		Expect(groups).To(HaveLen(1))
		Expect(groups[0].Name).To(Equal("P4"))
		Expect(groups[0].PRNumbers).To(ConsistOf(5))
	})

	It("skips draft PRs", func() {
		prs := []StuckPR{
			{Number: 6, Labels: []string{"lgtm", "approved"}, IsDraft: true},
		}
		groups := classify(prs)
		Expect(groups).To(BeEmpty())
	})

	It("skips work-in-progress PRs", func() {
		prs := []StuckPR{
			{Number: 7, Labels: []string{"lgtm", "approved", "do-not-merge/work-in-progress"}},
		}
		groups := classify(prs)
		Expect(groups).To(BeEmpty())
	})

	It("distributes mixed PRs across all priority groups", func() {
		prs := []StuckPR{
			{Number: 10, Labels: []string{"lgtm", "approved"}},
			{Number: 20, Labels: []string{"lgtm"}},
			{Number: 30, Labels: []string{"size/S"}},
			{Number: 40, Labels: []string{"do-not-merge/hold"}},
			{Number: 50, Labels: []string{"lgtm", "approved"}, IsDraft: true},
		}
		groups := classify(prs)
		Expect(groups).To(HaveLen(4))
		Expect(groups[0].Name).To(Equal("P1"))
		Expect(groups[0].PRNumbers).To(ConsistOf(10))
		Expect(groups[1].Name).To(Equal("P2"))
		Expect(groups[1].PRNumbers).To(ConsistOf(20))
		Expect(groups[2].Name).To(Equal("P3"))
		Expect(groups[2].PRNumbers).To(ConsistOf(30))
		Expect(groups[3].Name).To(Equal("P4"))
		Expect(groups[3].PRNumbers).To(ConsistOf(40))
	})

	It("sorts PR numbers within each group", func() {
		prs := []StuckPR{
			{Number: 30, Labels: []string{"lgtm", "approved"}},
			{Number: 10, Labels: []string{"lgtm", "approved"}},
			{Number: 20, Labels: []string{"lgtm", "approved"}},
		}
		groups := classify(prs)
		Expect(groups).To(HaveLen(1))
		Expect(groups[0].PRNumbers).To(Equal([]int{10, 20, 30}))
	})

	It("returns empty groups for empty input", func() {
		groups := classify(nil)
		Expect(groups).To(BeEmpty())
	})

	It("recognizes approved- prefix variants for P1", func() {
		prs := []StuckPR{
			{Number: 8, Labels: []string{"lgtm", "approved-for-merge"}},
		}
		groups := classify(prs)
		Expect(groups).To(HaveLen(1))
		Expect(groups[0].Name).To(Equal("P1"))
	})
})

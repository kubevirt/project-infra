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

var _ = Describe("splitBatches", func() {

	DescribeTable("splits items into batches of given size",
		func(items []int, size int, expected [][]int) {
			Expect(splitBatches(items, size)).To(Equal(expected))
		},
		Entry("exact split",
			[]int{1, 2, 3, 4}, 2,
			[][]int{{1, 2}, {3, 4}},
		),
		Entry("remainder batch",
			[]int{1, 2, 3, 4, 5}, 2,
			[][]int{{1, 2}, {3, 4}, {5}},
		),
		Entry("single batch when size >= len",
			[]int{1, 2, 3}, 10,
			[][]int{{1, 2, 3}},
		),
		Entry("batch size of 1",
			[]int{1, 2, 3}, 1,
			[][]int{{1}, {2}, {3}},
		),
		Entry("empty input",
			[]int{}, 5,
			nil,
		),
		Entry("nil input",
			nil, 5,
			nil,
		),
	)
})

var _ = Describe("collectPRNumbers", func() {

	groups := []PriorityGroup{
		{Name: "P1", PRNumbers: []int{10, 20}},
		{Name: "P2", PRNumbers: []int{30}},
		{Name: "P3", PRNumbers: []int{40, 50, 60}},
	}

	It("collects all PR numbers when no filter is set", func() {
		Expect(collectPRNumbers(groups, "")).To(Equal([]int{10, 20, 30, 40, 50, 60}))
	})

	It("filters by group name", func() {
		Expect(collectPRNumbers(groups, "P1")).To(Equal([]int{10, 20}))
	})

	It("returns nil for non-existent group", func() {
		Expect(collectPRNumbers(groups, "P99")).To(BeNil())
	})

	It("returns nil for empty groups", func() {
		Expect(collectPRNumbers(nil, "")).To(BeNil())
	})
})

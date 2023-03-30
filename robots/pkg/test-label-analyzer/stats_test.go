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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package test_label_analyzer

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("GetStatsFromGinkgoOutline", func() {

	Context("w/o recursion", func() {

		It("does not match any test since no spec", func() {
			Expect(GetStatsFromGinkgoOutline(NewQuarantineDefaultConfig(),
				[]*GinkgoNode{
					{
						Spec: false,
					},
				})).To(BeEquivalentTo(
				&TestStats{
					SpecsTotal:    0,
					SpecsMatching: 0,
				}))
		})

		It("does not match any test since spec doesn't match", func() {
			Expect(GetStatsFromGinkgoOutline(
				NewQuarantineDefaultConfig(),
				[]*GinkgoNode{
					{
						Spec: true,
					},
				})).To(BeEquivalentTo(
				&TestStats{
					SpecsTotal:    1,
					SpecsMatching: 0,
				}))
		})

		It("does match test text", func() {
			Expect(GetStatsFromGinkgoOutline(
				NewQuarantineDefaultConfig(),
				[]*GinkgoNode{
					{
						Text: "[QUARANTINE]",
						Spec: true,
					},
				})).To(BeEquivalentTo(
				&TestStats{
					SpecsTotal:    1,
					SpecsMatching: 1,
					MatchingSpecPathes: [][]*GinkgoNode{
						{
							{
								Text: "[QUARANTINE]",
								Spec: true,
							},
						},
					},
				}))
		})

	})

	Context("w recursion", func() {

		It("has sub node which is a spec", func() {
			Expect(GetStatsFromGinkgoOutline(
				NewQuarantineDefaultConfig(),
				[]*GinkgoNode{
					{
						Nodes: []*GinkgoNode{
							{
								Spec: true,
							},
						},
					},
				})).To(BeEquivalentTo(
				&TestStats{
					SpecsTotal:    1,
					SpecsMatching: 0,
				}))
		})

		It("has sub node which is a spec", func() {
			Expect(GetStatsFromGinkgoOutline(
				NewQuarantineDefaultConfig(),
				[]*GinkgoNode{
					{
						Nodes: []*GinkgoNode{
							{
								Text: "[QUARANTINE]",
								Spec: true,
							},
						},
					},
				})).To(BeEquivalentTo(
				&TestStats{
					SpecsTotal:    1,
					SpecsMatching: 1,
					MatchingSpecPathes: [][]*GinkgoNode{
						{
							{},
							{
								Text: "[QUARANTINE]",
								Spec: true,
							},
						},
					},
				}))
		})

		It("collects the test names", func() {
			Expect(GetStatsFromGinkgoOutline(
				NewQuarantineDefaultConfig(),
				[]*GinkgoNode{
					{
						Text: "parent",
						Nodes: []*GinkgoNode{
							{
								Text: "child",
								Spec: false,
								Nodes: []*GinkgoNode{
									{
										Text: "[QUARANTINE]",
										Spec: true,
									},
								},
							},
						},
					},
				})).To(BeEquivalentTo(
				&TestStats{
					SpecsTotal:    1,
					SpecsMatching: 1,
					MatchingSpecPathes: [][]*GinkgoNode{
						{
							{
								Text: "parent",
							},
							{
								Text: "child",
								Spec: false,
							},
							{
								Text: "[QUARANTINE]",
								Spec: true,
							},
						},
					},
				}))
		})

		It("collects the test names if parent contains the matching label", func() {
			Expect(GetStatsFromGinkgoOutline(
				NewQuarantineDefaultConfig(),
				[]*GinkgoNode{
					{
						Text: "parent",
						Nodes: []*GinkgoNode{
							{
								Text: "[QUARANTINE]",
								Spec: false,
								Nodes: []*GinkgoNode{
									{
										Text: "child",
										Spec: true,
									},
								},
							},
						},
					},
				})).To(BeEquivalentTo(
				&TestStats{
					SpecsTotal:    1,
					SpecsMatching: 1,
					MatchingSpecPathes: [][]*GinkgoNode{
						{
							{
								Text: "parent",
							},
							{
								Text: "[QUARANTINE]",
								Spec: false,
							},
							{
								Text: "child",
								Spec: true,
							},
						},
					},
				}))
		})

		It("doesnt collect the test names twice if parent and child contain the matching label", func() {
			Expect(GetStatsFromGinkgoOutline(
				NewQuarantineDefaultConfig(),
				[]*GinkgoNode{
					{
						Text: "parent",
						Nodes: []*GinkgoNode{
							{
								Text: "[QUARANTINE]",
								Spec: false,
								Nodes: []*GinkgoNode{
									{
										Text: "[QUARANTINE]",
										Spec: true,
									},
								},
							},
						},
					},
				})).To(BeEquivalentTo(
				&TestStats{
					SpecsTotal:    1,
					SpecsMatching: 1,
					MatchingSpecPathes: [][]*GinkgoNode{
						{
							{
								Text: "parent",
							},
							{
								Text: "[QUARANTINE]",
								Spec: false,
							},
							{
								Text: "[QUARANTINE]",
								Spec: true,
							},
						},
					},
				}))
		})

		It("does collect the test nodes twice if parent contains the matching label", func() {
			Expect(GetStatsFromGinkgoOutline(
				NewQuarantineDefaultConfig(),
				[]*GinkgoNode{
					{
						Text: "parent",
						Nodes: []*GinkgoNode{
							{
								Text: "[QUARANTINE]",
								Spec: false,
								Nodes: []*GinkgoNode{
									{
										Text: "first",
										Spec: true,
									},
									{
										Text: "second",
										Spec: true,
									},
								},
							},
						},
					},
				})).To(BeEquivalentTo(
				&TestStats{
					SpecsTotal:    2,
					SpecsMatching: 2,
					MatchingSpecPathes: [][]*GinkgoNode{
						{
							{
								Text: "parent",
							},
							{
								Text: "[QUARANTINE]",
								Spec: false,
							},
							{
								Text: "first",
								Spec: true,
							},
						},
						{
							{
								Text: "parent",
							},
							{
								Text: "[QUARANTINE]",
								Spec: false,
							},
							{
								Text: "second",
								Spec: true,
							},
						},
					},
				}))
		})

	})

})

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
	"kubevirt.io/project-infra/robots/pkg/ginkgo"
)

var _ = Describe("GetStatsFromGinkgoOutline", func() {

	var quarantineLabelCategory *LabelCategory

	BeforeEach(func() {
		quarantineLabelCategory = NewQuarantineLabelCategory()
	})

	Context("w/o recursion", func() {

		It("does not match any test since no spec", func() {
			Expect(GetStatsFromGinkgoOutline(NewQuarantineDefaultConfig(),
				[]*ginkgo.Node{
					{
						Spec: false,
					},
				})).To(BeEquivalentTo(
				&TestStats{
					SpecsTotal: 0,
				}))
		})

		It("does not match any test since spec doesn't match", func() {
			Expect(GetStatsFromGinkgoOutline(
				NewQuarantineDefaultConfig(),
				[]*ginkgo.Node{
					{
						Spec: true,
					},
				})).To(BeEquivalentTo(
				&TestStats{
					SpecsTotal: 1,
				}))
		})

		It("does match test text", func() {
			quarantineLabelCategory.Hits = 1
			Expect(GetStatsFromGinkgoOutline(
				NewQuarantineDefaultConfig(),
				[]*ginkgo.Node{
					{
						Text: "[QUARANTINE]",
						Spec: true,
					},
				})).To(BeEquivalentTo(
				&TestStats{
					SpecsTotal: 1,
					MatchingSpecPaths: []*PathStats{
						{
							Path: []*ginkgo.Node{
								{
									Text: "[QUARANTINE]",
									Spec: true,
								},
							},
							MatchingCategory: quarantineLabelCategory,
						},
					},
				}))
		})

	})

	Context("w recursion", func() {

		It("has sub node which is a spec", func() {
			Expect(GetStatsFromGinkgoOutline(
				NewQuarantineDefaultConfig(),
				[]*ginkgo.Node{
					{
						Nodes: []*ginkgo.Node{
							{
								Spec: true,
							},
						},
					},
				})).To(BeEquivalentTo(
				&TestStats{
					SpecsTotal: 1,
				}))
		})

		It("has sub node which is a spec", func() {
			quarantineLabelCategory.Hits = 1
			Expect(GetStatsFromGinkgoOutline(
				NewQuarantineDefaultConfig(),
				[]*ginkgo.Node{
					{
						Nodes: []*ginkgo.Node{
							{
								Text: "[QUARANTINE]",
								Spec: true,
							},
						},
					},
				})).To(BeEquivalentTo(
				&TestStats{
					SpecsTotal: 1,
					MatchingSpecPaths: []*PathStats{
						{
							Path: []*ginkgo.Node{
								{},
								{
									Text: "[QUARANTINE]",
									Spec: true,
								},
							},
							MatchingCategory: quarantineLabelCategory,
						},
					},
				}))
		})

		It("collects the test names", func() {
			quarantineLabelCategory.Hits = 1
			Expect(GetStatsFromGinkgoOutline(
				NewQuarantineDefaultConfig(),
				[]*ginkgo.Node{
					{
						Text: "parent",
						Nodes: []*ginkgo.Node{
							{
								Text: "child",
								Spec: false,
								Nodes: []*ginkgo.Node{
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
					SpecsTotal: 1,
					MatchingSpecPaths: []*PathStats{
						{
							Path: []*ginkgo.Node{
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
							MatchingCategory: quarantineLabelCategory,
						},
					}}))
		})

		It("collects the test names if parent contains the matching label", func() {
			quarantineLabelCategory.Hits = 1
			Expect(GetStatsFromGinkgoOutline(
				NewQuarantineDefaultConfig(),
				[]*ginkgo.Node{
					{
						Text: "parent",
						Nodes: []*ginkgo.Node{
							{
								Text: "[QUARANTINE]",
								Spec: false,
								Nodes: []*ginkgo.Node{
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
					SpecsTotal: 1,
					MatchingSpecPaths: []*PathStats{
						{
							Path: []*ginkgo.Node{
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
							MatchingCategory: quarantineLabelCategory,
						},
					}}))
		})

		It("doesnt collect the test names twice if parent and child contain the matching label", func() {
			quarantineLabelCategory.Hits = 1
			Expect(GetStatsFromGinkgoOutline(
				NewQuarantineDefaultConfig(),
				[]*ginkgo.Node{
					{
						Text: "parent",
						Nodes: []*ginkgo.Node{
							{
								Text: "[QUARANTINE]",
								Spec: false,
								Nodes: []*ginkgo.Node{
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
					SpecsTotal: 1,
					MatchingSpecPaths: []*PathStats{
						{
							Path: []*ginkgo.Node{
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
							MatchingCategory: quarantineLabelCategory,
						},
					}}))
		})

		It("does collect the test nodes twice if parent contains the matching label", func() {
			quarantineLabelCategory.Hits = 2
			Expect(GetStatsFromGinkgoOutline(
				NewQuarantineDefaultConfig(),
				[]*ginkgo.Node{
					{
						Text: "parent",
						Nodes: []*ginkgo.Node{
							{
								Text: "[QUARANTINE]",
								Spec: false,
								Nodes: []*ginkgo.Node{
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
					SpecsTotal: 2,
					MatchingSpecPaths: []*PathStats{
						{
							Path: []*ginkgo.Node{
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
							MatchingCategory: quarantineLabelCategory,
						},
						{
							Path: []*ginkgo.Node{
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
							MatchingCategory: quarantineLabelCategory,
						}},
				}))
		})

		It("does collect the test node if the expression would match more than a single node", func() {
			category := NewPartialTestNameLabelCategory("parent first child")
			category.Hits = 2
			Expect(GetStatsFromGinkgoOutline(
				NewTestNameDefaultConfig("parent first child"),
				[]*ginkgo.Node{
					{
						Text: "parent",
						Nodes: []*ginkgo.Node{
							{
								Text: "first child",
								Spec: false,
								Nodes: []*ginkgo.Node{
									{
										Text: "child of first child",
										Spec: true,
									},
									{
										Text: "second child of first child",
										Spec: true,
									},
								},
							},
						},
					},
				})).To(BeEquivalentTo(
				&TestStats{
					SpecsTotal: 2,
					MatchingSpecPaths: []*PathStats{
						{
							Path: []*ginkgo.Node{
								{
									Text: "parent",
								},
								{
									Text: "first child",
								},
								{
									Text: "child of first child",
									Spec: true,
								},
							},
							MatchingCategory: category,
						},
						{
							Path: []*ginkgo.Node{
								{
									Text: "parent",
								},
								{
									Text: "first child",
								},
								{
									Text: "second child of first child",
									Spec: true,
								},
							},
							MatchingCategory: category,
						},
					}}))
		})

	})

})

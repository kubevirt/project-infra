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
	"time"
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
					SpecsTotal: 0,
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
					SpecsTotal: 1,
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
					SpecsTotal: 1,
					MatchingSpecPaths: []*PathStats{
						{
							Path: []*GinkgoNode{
								{
									Text: "[QUARANTINE]",
									Spec: true,
								},
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
					SpecsTotal: 1,
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
					SpecsTotal: 1,
					MatchingSpecPaths: []*PathStats{
						{
							Path: []*GinkgoNode{
								{},
								{
									Text: "[QUARANTINE]",
									Spec: true,
								},
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
					SpecsTotal: 1,
					MatchingSpecPaths: []*PathStats{
						{
							Path: []*GinkgoNode{
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
					}}))
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
					SpecsTotal: 1,
					MatchingSpecPaths: []*PathStats{
						{
							Path: []*GinkgoNode{
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
					}}))
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
					SpecsTotal: 1,
					MatchingSpecPaths: []*PathStats{
						{
							Path: []*GinkgoNode{
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
					}}))
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
					SpecsTotal: 2,
					MatchingSpecPaths: []*PathStats{
						{
							Path: []*GinkgoNode{
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
							}},
						{
							Path: []*GinkgoNode{
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
						}},
				}))
		})

		It("does collect the test node if the expression would match more than a single node", func() {
			Expect(GetStatsFromGinkgoOutline(
				NewTestNameDefaultConfig("parent first child"),
				[]*GinkgoNode{
					{
						Text: "parent",
						Nodes: []*GinkgoNode{
							{
								Text: "first child",
								Spec: false,
								Nodes: []*GinkgoNode{
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
							Path: []*GinkgoNode{
								{
									Text: "parent",
								},
								{
									Text: "first child",
									Spec: false,
								},
								{
									Text: "child of first child",
									Spec: true,
								},
							}},
						{Path: []*GinkgoNode{
							{
								Text: "parent",
							},
							{
								Text: "first child",
								Spec: false,
							},
							{
								Text: "second child of first child",
								Spec: true,
							},
						},
						},
					}}))
		})

	})

})

var _ = Describe("Extract git info", func() {

	DescribeTable("parse regex",
		func(valueToParse string) {
			strings := gitBlameRegex.FindAllStringSubmatch(valueToParse, -1)
			Expect(strings).To(HaveLen(1))
			Expect(strings[0]).To(HaveLen(7))
		},
		Entry("basic 1", "749cf0488 (Ben Oukhanov 2023-02-15 18:24:49 +0200  26) var _ = Describe(\"VM Console Proxy Operand\", func() {"),
		Entry("basic 2", `a32051d928 tests/operator_test.go     (fossedihelm   2022-11-30 09:45:57 +0100  111) var _ = Describe("[Serial][sig-operator]Operator", Serial, decorators.SigOperator, func() {`),
	)

	Context("blame", func() {

		var lines = []string{
			`749cf0488 (Ben Oukhanov 2023-02-15 18:24:49 +0200  26) var _ = Describe("VM Console Proxy Operand", func() {`,
			`749cf0488 (Ben Oukhanov 2023-02-15 18:24:49 +0200 179) \tContext("Resource change", func() {`,
			`749cf0488 (Ben Oukhanov 2023-02-15 18:24:49 +0200 208) \t\tDescribeTable("should restore modified app labels", expectAppLabelsRestoreAfterUpdate,`,
			`749cf0488 (Ben Oukhanov 2023-02-15 18:24:49 +0200 213) \t\t\tEntry("[test_id:TODO] deployment", \u0026deploymentResource),`,
		}
		var expectedCommitID = "749cf0488"
		var expectedAuthor = "Ben Oukhanov"
		var expectedDate time.Time
		var expectedLineNo = 26
		var expectedLine = "var _ = Describe(\"VM Console Proxy Operand\", func() {"

		BeforeEach(func() {
			expectedDate, _ = time.Parse(gitDateLayout, "2023-02-15 18:24:49 +0200")
		})

		It("extracts info", func() {
			Expect(ExtractGitBlameInfo(lines)).ToNot(BeNil())
		})

		It("fills fields", func() {
			gitBlameInfo := ExtractGitBlameInfo(lines)[0]
			Expect(gitBlameInfo.CommitID).To(BeEquivalentTo(expectedCommitID))
			Expect(gitBlameInfo.Author).To(BeEquivalentTo(expectedAuthor))
			Expect(gitBlameInfo.Date).To(BeEquivalentTo(expectedDate))
			Expect(gitBlameInfo.LineNo).To(BeEquivalentTo(expectedLineNo))
			Expect(gitBlameInfo.Line).To(BeEquivalentTo(expectedLine))
		})

		It("extracts from other lines", func() {
			errLines := []string{
				`a32051d928 tests/operator_test.go     (fossedihelm   2022-11-30 09:45:57 +0100  111) var _ = Describe("[Serial][sig-operator]Operator", Serial, decorators.SigOperator, func() {`,
				`eb6dc428cd tests/operator_test.go     (Igor Bezukh   2022-03-15 18:18:47 +0200 2928) 	Context("Obsolete ConfigMap", func() {`,
				`52e637192d tests/operator/operator.go (Daniel Hiller 2023-05-26 10:21:35 +0200 2942) 		It("[QUARANTINE] should emit event if the obsolete kubevirt-config configMap still exists", func() {`,
			}
			blameInfo := ExtractGitBlameInfo(errLines)
			Expect(blameInfo).ToNot(BeNil())
			gitBlameInfo := blameInfo[0]
			Expect(gitBlameInfo.CommitID).To(BeEquivalentTo("a32051d928"))
			Expect(gitBlameInfo.Author).To(BeEquivalentTo("fossedihelm"))
			expectedDate2, _ := time.Parse(gitDateLayout, "2022-11-30 09:45:57 +0100")
			Expect(gitBlameInfo.Date).To(BeEquivalentTo(expectedDate2))
			Expect(gitBlameInfo.LineNo).To(BeEquivalentTo(111))
			Expect(gitBlameInfo.Line).To(BeEquivalentTo(`var _ = Describe("[Serial][sig-operator]Operator", Serial, decorators.SigOperator, func() {`))
		})

		PIt("TODO:naming extracts from further lines", func() {
			errLines := []string{
				`0df3f3c5129 (João Vilaça 2023-03-22 10:45:53 +0000 55) var _ = Describe("[Serial][sig-monitoring]VM Monitoring", Serial, decorators.SigMonitoring, func() {`,
				`ba2fdf5f25a (João Vilaça 2023-05-11 10:45:18 +0100 63) 	Context("Cluster VM metrics", func() {`,
				`ba2fdf5f25a (João Vilaça 2023-05-11 10:45:18 +0100 64) 		It("kubevirt_number_of_vms should reflect the number of VMs", func() {`,
				``,
			}
			blameInfo := ExtractGitBlameInfo(errLines)
			Expect(blameInfo).ToNot(BeNil())
			gitBlameInfo := blameInfo[0]
			Expect(gitBlameInfo.CommitID).To(BeEquivalentTo("0df3f3c5129"))
			Expect(gitBlameInfo.Author).To(BeEquivalentTo("João Vilaça"))
			expectedDate2, _ := time.Parse(gitDateLayout, "2023-05-11 10:45:18 +0100")
			Expect(gitBlameInfo.Date).To(BeEquivalentTo(expectedDate2))
			Expect(gitBlameInfo.LineNo).To(BeEquivalentTo(111))
			Expect(gitBlameInfo.Line).To(BeEquivalentTo(`var _ = Describe("[Serial][sig-monitoring]VM Monitoring", Serial, decorators.SigMonitoring, func() {`))
		})
	})
})

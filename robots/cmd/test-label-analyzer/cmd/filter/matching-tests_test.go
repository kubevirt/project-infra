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
 */

package filter

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/project-infra/robots/pkg/ginkgo"
	"kubevirt.io/project-infra/robots/pkg/git"
	testlabelanalyzer "kubevirt.io/project-infra/robots/pkg/test-label-analyzer"
	"os"
	"path/filepath"
	"time"
)

var _ = Describe("filter", func() {
	Context("matching tests", func() {
		DescribeTable("are filtered",
			func(input testlabelanalyzer.TestFilesStats, expected matchingTests) {
				Expect(filterMatchingTests(input, "")).To(BeEquivalentTo(expected))
			},
			Entry("empty input", testlabelanalyzer.TestFilesStats{}, nil),
			Entry("simple input",
				testlabelanalyzer.TestFilesStats{
					FilesStats: []*testlabelanalyzer.FileStats{
						{
							TestStats: &testlabelanalyzer.TestStats{
								SpecsTotal: 0,
								MatchingSpecPaths: []*testlabelanalyzer.PathStats{
									{
										Path: []*ginkgo.Node{
											{
												Text: "VM Live Migration",
											},
											{
												Text: "[Serial][QUARANTINE] with a dedicated migration network",
											},
											{
												Text: "Should migrate over that network",
											},
										},
										MatchingCategory: &testlabelanalyzer.LabelCategory{
											Name:            "flaky test - Tracked in https://github.com/kubevirt/kubevirt/issues/37",
											TestNameLabelRE: testlabelanalyzer.NewRegexp("with a dedicated migration network Should migrate over that network"),
											BlameLine: &git.BlameLine{
												CommitID: "",
												Author:   "Daniel Hiller",
												Date:     mustParseDate("2023-07-20T13:24:47+02:00"),
												LineNo:   0,
												Line:     "",
											},
											Hits: 1,
										},
									},
								},
							},
							RemoteURL: "",
						},
					},
				},
				matchingTests{
					matchingTest{
						Id:       "with a dedicated migration network Should migrate over that network",
						Reason:   "flaky test - Tracked in https://github.com/kubevirt/kubevirt/issues/37",
						Version:  "",
						TestName: "VM Live Migration [Serial][QUARANTINE] with a dedicated migration network Should migrate over that network",
						BlameLine: &git.BlameLine{
							CommitID: "",
							Author:   "Daniel Hiller",
							Date:     mustParseDate("2023-07-20T13:24:47+02:00"),
							LineNo:   0,
							Line:     "",
						},
					},
				},
			),
		)

		It("filters file successfully", func() {
			temp, err := os.MkdirTemp("", "")
			Expect(err).ToNot(HaveOccurred())

			filterStatsMatchingTestsOpts = &filterStatsMatchingTestsOptions{
				outputFilePath: filepath.Join(temp, "filtered-output.json"),
			}

			Expect(filterMatchingTestsFromInputFile("testdata/stats-output.json", filterStatsMatchingTestsOpts)).To(Succeed())
		})
	})
})

func mustParseDate(date string) time.Time {
	parse, err := time.Parse(time.RFC3339, date)
	Expect(err).ToNot(HaveOccurred())
	return parse
}

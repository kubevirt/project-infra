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

package cmd

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/project-infra/robots/pkg/git"
	test_label_analyzer "kubevirt.io/project-infra/robots/pkg/test-label-analyzer"
	"time"
)

var _ = Describe("stats", func() {

	Context("NewStatsHTMLData", func() {

		const remoteURL = "http://github.com/dhiller/test"
		var simpleQuarantineConfig = test_label_analyzer.NewTestNameDefaultConfig("[QUARANTINE]")

		It("returns data from file stats", func() {
			// t.MatchingPath.MatchingCategory
			Expect(NewStatsHTMLData([]*test_label_analyzer.FileStats{
				{
					TestStats: &test_label_analyzer.TestStats{
						SpecsTotal: 2,
						MatchingSpecPaths: []*test_label_analyzer.PathStats{
							{
								Lines: nil,
								GitBlameLines: []*git.BlameLine{
									{
										CommitID: "1742",
										Author:   "johndoe@wherever.net",
										Date:     parseTime("2023-03-02T17:42:37Z"),
										LineNo:   0,
										Line:     "[QUARANTINE]",
									},
								},
								Path:             nil,
								MatchingCategory: &test_label_analyzer.LabelCategory{},
							},
						},
					},
					RemoteURL: remoteURL,
				},
			}, ConfigOptions{}).TestHTMLData).ToNot(BeEmpty())
		})

		PIt("sorts data by date for matching line", func() { // TODO: need to repair the comparison, seems the regexp has state that hinders it
			Expect(NewStatsHTMLData([]*test_label_analyzer.FileStats{
				{
					TestStats: &test_label_analyzer.TestStats{
						SpecsTotal: 2,
						MatchingSpecPaths: []*test_label_analyzer.PathStats{
							{
								Lines: nil,
								GitBlameLines: []*git.BlameLine{
									newGitBlameInfo(parseTime("2023-03-02T17:42:37Z"), "[QUARANTINE]"),
								},
								Path: nil,
							},
							{
								Lines: nil,
								GitBlameLines: []*git.BlameLine{
									newGitBlameInfo(parseTime("2023-02-02T17:42:37Z"), "[QUARANTINE]"),
								},
								Path: nil,
							},
						},
					},
					RemoteURL: remoteURL,
				},
			}, ConfigOptions{}).TestHTMLData).To(BeEquivalentTo(
				&StatsHTMLData{
					TestHTMLData: []*TestHTMLData{
						{
							Config: simpleQuarantineConfig,
							MatchingPath: &test_label_analyzer.PathStats{
								GitBlameLines: []*git.BlameLine{
									newGitBlameInfo(parseTime("2023-02-02T17:42:37Z"), "[QUARANTINE]"),
								},
							},
							RemoteURL: remoteURL,
						},
						{
							Config: simpleQuarantineConfig,
							MatchingPath: &test_label_analyzer.PathStats{
								GitBlameLines: []*git.BlameLine{
									newGitBlameInfo(parseTime("2023-03-02T17:42:37Z"), "[QUARANTINE]"),
								},
							},
							RemoteURL: remoteURL,
						},
					},
				}))
		})

	})

})

func newGitBlameInfo(t time.Time, line string) *git.BlameLine {
	return &git.BlameLine{
		CommitID: "1742",
		Author:   "johndoe@wherever.net",
		Date:     t,
		LineNo:   0,
		Line:     line,
	}
}

func parseTime(datetime string) time.Time {
	parse, err := time.Parse(time.RFC3339, datetime)
	if err != nil {
		panic(err)
	}
	return parse
}

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
	test_label_analyzer "kubevirt.io/project-infra/robots/pkg/test-label-analyzer"
	"time"
)

var _ = Describe("cmd/stats", func() {

	Context("getGinkgoOutlineFromFile", func() {

		PIt("generates outline from file", func() {
			outline, err := getGinkgoOutlineFromFile("testdata/simple_test.go")
			Expect(err).To(BeNil())
			Expect(outline).ToNot(BeNil())
		})

	})

	Context("NewStatsHTMLData", func() {

		const remoteURL = "http://github.com/dhiller/test"
		var simpleQuarantineConfig = test_label_analyzer.NewTestNameDefaultConfig("[QUARANTINE]")

		It("returns data from file stats", func() {
			Expect(NewStatsHTMLData([]*test_label_analyzer.FileStats{
				{
					Config: simpleQuarantineConfig,
					TestStats: &test_label_analyzer.TestStats{
						SpecsTotal: 2,
						MatchingSpecPaths: []*test_label_analyzer.PathStats{
							{
								Lines: nil,
								GitBlameLines: []*test_label_analyzer.GitBlameInfo{
									{
										CommitID: "1742",
										Author:   "johndoe@wherever.net",
										Date:     parseTime("2023-03-02T17:42:37Z"),
										LineNo:   0,
										Line:     "[QUARANTINE]",
									},
								},
								Path: nil,
							},
						},
					},
					RemoteURL: remoteURL,
				},
			}).TestHTMLData).ToNot(BeEmpty())
		})

		PIt("sorts data by date for matching line", func() { // TODO: need to repair the comparison, seems the regexp has state that hinders it
			Expect(NewStatsHTMLData([]*test_label_analyzer.FileStats{
				{
					Config: simpleQuarantineConfig,
					TestStats: &test_label_analyzer.TestStats{
						SpecsTotal: 2,
						MatchingSpecPaths: []*test_label_analyzer.PathStats{
							{
								Lines: nil,
								GitBlameLines: []*test_label_analyzer.GitBlameInfo{
									newGitBlameInfo(parseTime("2023-03-02T17:42:37Z"), "[QUARANTINE]"),
								},
								Path: nil,
							},
							{
								Lines: nil,
								GitBlameLines: []*test_label_analyzer.GitBlameInfo{
									newGitBlameInfo(parseTime("2023-02-02T17:42:37Z"), "[QUARANTINE]"),
								},
								Path: nil,
							},
						},
					},
					RemoteURL: remoteURL,
				},
			}).TestHTMLData).To(BeEquivalentTo(
				&StatsHTMLData{
					TestHTMLData: []*TestHTMLData{
						{
							Config: simpleQuarantineConfig,
							GitBlameLines: []*test_label_analyzer.GitBlameInfo{
								newGitBlameInfo(parseTime("2023-02-02T17:42:37Z"), "[QUARANTINE]"),
							},
							RemoteURL: remoteURL,
						},
						{
							Config: simpleQuarantineConfig,
							GitBlameLines: []*test_label_analyzer.GitBlameInfo{
								newGitBlameInfo(parseTime("2023-03-02T17:42:37Z"), "[QUARANTINE]"),
							},
							RemoteURL: remoteURL,
						},
					},
				}))
		})

	})

})

func newGitBlameInfo(t time.Time, line string) *test_label_analyzer.GitBlameInfo {
	return &test_label_analyzer.GitBlameInfo{
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

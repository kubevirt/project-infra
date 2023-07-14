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

package cmd

import (
	"fmt"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/project-infra/robots/pkg/git"
	test_label_analyzer "kubevirt.io/project-infra/robots/pkg/test-label-analyzer"
	"regexp"
	"time"
)

var _ = Describe("root tests", func() {

	Context("getConfig", func() {

		DescribeTable("returns a config",
			func(options *ConfigOptions, expectedConfig *test_label_analyzer.Config, expectedErr error) {
				config, err := options.getConfig()
				if err != nil {
					Expect(err).To(BeEquivalentTo(expectedErr))
				} else {
					Expect(config).To(BeEquivalentTo(expectedConfig))
					Expect(err).ToNot(HaveOccurred())
				}
			},
			Entry("returns err if no config selected",
				&ConfigOptions{
					ConfigFile:         "",
					ConfigName:         "",
					ginkgoOutlinePaths: nil,
					testFilePath:       "",
					remoteURL:          "",
					testNameLabelRE:    "",
					outputHTML:         false,
				},
				nil,
				fmt.Errorf("no configuration found!"),
			),
			Entry("for simple RE",
				&ConfigOptions{
					ConfigFile:         "",
					ConfigName:         "",
					ginkgoOutlinePaths: nil,
					testFilePath:       "",
					remoteURL:          "",
					testNameLabelRE:    "test regex",
					outputHTML:         false,
				},
				test_label_analyzer.NewTestNameDefaultConfig("test regex"),
				nil,
			),
			Entry("for quarantine config",
				&ConfigOptions{
					ConfigFile:         "",
					ConfigName:         "quarantine",
					ginkgoOutlinePaths: nil,
					testFilePath:       "",
					remoteURL:          "",
					testNameLabelRE:    "",
					outputHTML:         false,
				},
				test_label_analyzer.NewQuarantineDefaultConfig(),
				nil,
			),
			Entry("for file with test names",
				&ConfigOptions{
					ConfigFile:          "",
					FilterTestNamesFile: "testdata/filter-test-names.json",
					ConfigName:          "",
					ginkgoOutlinePaths:  nil,
					testFilePath:        "",
					remoteURL:           "",
					testNameLabelRE:     "",
					outputHTML:          false,
				},
				&test_label_analyzer.Config{
					Categories: []*test_label_analyzer.LabelCategory{
						{
							Name:            "flaky",
							TestNameLabelRE: test_label_analyzer.NewRegexp("test name 1"),
							GinkgoLabelRE:   nil,
							BlameLine: &git.BlameLine{
								CommitID: "716b4b979",
								Author:   "Daniel Hiller",
								Date:     mustParseDate("2023-07-12T17:12:11+02:00"),
								LineNo:   3,
								Line:     `    "id": "test name 1",`,
							},
						},
						{
							Name:            "also flaky",
							TestNameLabelRE: test_label_analyzer.NewRegexp("test name 2"),
							GinkgoLabelRE:   nil,
							BlameLine: &git.BlameLine{
								CommitID: "716b4b979",
								Author:   "Daniel Hiller",
								Date:     mustParseDate("2023-07-12T17:12:11+02:00"),
								LineNo:   7,
								Line:     `    "id": "test name 2",`,
							},
						},
						{
							Name:            "also flaky",
							TestNameLabelRE: test_label_analyzer.NewRegexp(regexp.QuoteMeta("[sig-compute]test name 3")),
							GinkgoLabelRE:   nil,
							BlameLine: &git.BlameLine{
								CommitID: "716b4b979",
								Author:   "Daniel Hiller",
								Date:     mustParseDate("2023-07-12T17:12:11+02:00"),
								LineNo:   11,
								Line:     `    "id": "[sig-compute]test name 3",`,
							},
						},
					},
				},
				nil,
			),
		)
	})
})

func mustParseDate(date string) time.Time {
	parse, err := time.Parse(time.RFC3339, date)
	Expect(err).ToNot(HaveOccurred())
	return parse
}

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
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"kubevirt.io/project-infra/pkg/git"
	testlabelanalyzer "kubevirt.io/project-infra/pkg/test-label-analyzer"
)

var _ = Describe("root tests", func() {

	Context("getConfig", func() {

		DescribeTable("returns a config",
			func(options *ConfigOptions, expectedConfig *testlabelanalyzer.Config, expectedErr error) {
				config, err := options.getConfig()
				if expectedErr != nil {
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
				fmt.Errorf("no configuration found"),
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
				testlabelanalyzer.NewTestNameDefaultConfig("test regex"),
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
				testlabelanalyzer.NewQuarantineDefaultConfig(),
				nil,
			),
			Entry("for config name that doesn't exist",
				&ConfigOptions{
					ConfigFile:         "",
					ConfigName:         "ihavenoclue",
					ginkgoOutlinePaths: nil,
					testFilePath:       "",
					remoteURL:          "",
					testNameLabelRE:    "",
					outputHTML:         false,
				},
				nil,
				fmt.Errorf("config \"ihavenoclue\" does not exist"),
			),
		)

		// FIXME
		/*
			[FAILED] Unexpected error:
			    <*errors.errorString | 0xc0001ca330>:
			    exec /usr/bin/git blame filter-test-names.json failed: fatal: no such ref: HEAD

			    {
			        s: "exec /usr/bin/git blame filter-test-names.json failed: fatal: no such ref: HEAD\n",
			    }
			occurred
		*/
		PIt("loads for file with test names", func() {
			var tempDir string
			var err error

			tempDir, err = os.MkdirTemp("", "")
			Expect(err).ToNot(HaveOccurred())
			command := exec.Command("git", "init")
			command.Dir = tempDir
			err = command.Run()
			Expect(err).ToNot(HaveOccurred())

			const gitTestFileName = "testdata/filter-test-names.json"
			var file []byte
			file, err = os.ReadFile(gitTestFileName)
			Expect(err).ToNot(HaveOccurred())
			targetFile := filepath.Join(tempDir, path.Base(gitTestFileName))
			err = os.WriteFile(targetFile, file, 0666)
			Expect(err).ToNot(HaveOccurred())

			command = exec.Command("git", "add", path.Base(gitTestFileName))
			command.Dir = tempDir
			err = command.Run()
			Expect(err).ToNot(HaveOccurred())

			command = exec.Command("git", "commit", "-m", "test commit")
			command.Dir = tempDir
			err = command.Run()
			Expect(err).ToNot(HaveOccurred())

			var gitBlameLines []*git.BlameLine
			gitBlameLines, err = git.GetBlameLinesForFile(targetFile)
			Expect(err).ToNot(HaveOccurred())

			options := &ConfigOptions{
				ConfigFile:          "",
				FilterTestNamesFile: targetFile,
				ConfigName:          "",
				ginkgoOutlinePaths:  nil,
				testFilePath:        "",
				remoteURL:           "",
				testNameLabelRE:     "",
				outputHTML:          false,
			}
			expectedConfig := &testlabelanalyzer.Config{
				Categories: []*testlabelanalyzer.LabelCategory{
					{
						Name:            "flaky",
						TestNameLabelRE: testlabelanalyzer.NewRegexp("test name 1"),
						GinkgoLabelRE:   nil,
						BlameLine:       gitBlameLines[2],
					},
					{
						Name:            "also flaky",
						TestNameLabelRE: testlabelanalyzer.NewRegexp("test name 2"),
						GinkgoLabelRE:   nil,
						BlameLine:       gitBlameLines[6],
					},
					{
						Name:            "also flaky",
						TestNameLabelRE: testlabelanalyzer.NewRegexp(regexp.QuoteMeta("[sig-compute]test name 3")),
						GinkgoLabelRE:   nil,
						BlameLine:       gitBlameLines[10],
					},
				},
			}

			config, err := options.getConfig()
			Expect(config).To(BeEquivalentTo(expectedConfig))
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

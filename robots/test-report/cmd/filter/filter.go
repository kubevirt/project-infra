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

package filter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	testreport "kubevirt.io/project-infra/pkg/test-report"
)

// defaultFileOutputPattern defines the pattern that is used to generate the file names, to be output in target directory.
const defaultFileOutputPattern = "not-run-tests-%s-%s.txt"
const shortUsage = "Filters the output of test-report execution command into one text file per team and version, containing lists of not run tests"

var fileOutputPattern string
var outputDirectory string
var filterCmd = &cobra.Command{
	Use:   "filter",
	Short: shortUsage,
	Long: shortUsage + `.
In detail "not run" means, that per test there was no single occurrence of it being run, nor was it marked as unsupported by being part of the dont_run_tests.json. 

Usage:

	$ output_dir=$(mktemp -d) && \
		test-report filter \
			--output-directory $output_dir \
			"--output-pattern=/not-run-tests-%s-%s.txt" /path/to/test-report.json

Output will then be written to $output_dir, which contains a set of files that contain the test names of the test
that have not been run. I.e. given
	versions := ["4.11", "4.12"]
	groups := ["virtualization", "network"]

it would create files

	not-run-tests-4.11-virtualization.txt
	not-run-tests-4.12-virtualization.txt
	not-run-tests-4.11-network.txt
	not-run-tests-4.12-network.txt
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if outputDirectory == "" {
			tempDir, err := os.MkdirTemp("", "test-report-*")
			if err != nil {
				return err
			}
			outputDirectory = tempDir
			log.Printf("writing output to directory: %s", outputDirectory)
		} else {
			stat, err := os.Stat(outputDirectory)
			if os.IsNotExist(err) {
				return fmt.Errorf("directory %s does not exist: %v", outputDirectory, err)
			}
			if !stat.IsDir() {
				return fmt.Errorf("%s is not a directory", outputDirectory)
			}
		}
		if len(args) != 1 {
			return fmt.Errorf("no file as argument given")
		}
		_, err := os.Stat(args[0])
		if os.IsNotExist(err) {
			return err
		}
		file, err := os.ReadFile(args[0])
		if err != nil {
			return err
		}
		var input *map[string]map[string]int
		err = json.Unmarshal(file, &input)
		if err != nil {
			return err
		}
		filtered := runFilter(input, nil)
		for group, versions := range filtered {
			for version, tests := range versions {
				testsWithNewlines := strings.Join(tests, "\n")
				err := os.WriteFile(filepath.Join(outputDirectory, fmt.Sprintf(fileOutputPattern, version, group)), []byte(testsWithNewlines), os.ModePerm)
				if err != nil {
					return err
				}
			}
		}
		return nil
	},
}

func init() {
	filterCmd.PersistentFlags().StringVar(&fileOutputPattern, "output-pattern", defaultFileOutputPattern, "")
	filterCmd.PersistentFlags().StringVar(&outputDirectory, "output-directory", "", "")
}

func FilterCmd() *cobra.Command {
	return filterCmd
}

type groupConfig struct {
	name    string
	sig     *regexp.Regexp
	lanes   *regexp.Regexp
	version *regexp.Regexp
}

type groupConfigs []groupConfig

const versionPattern = ".*-(4\\.[0-9]{2}).*"

var defaultGroupConfigs = groupConfigs{
	groupConfig{
		name:    "storage",
		version: regexp.MustCompile(versionPattern),
		sig:     regexp.MustCompile(`.*\[sig-storage\].*`),
		lanes:   regexp.MustCompile(".*(storage|quarantined).*"),
	},
	groupConfig{
		name:    "virtualization",
		version: regexp.MustCompile(versionPattern),
		sig:     regexp.MustCompile(`.*\[sig-(compute|operator)\].*`),
		lanes:   regexp.MustCompile(".*(compute|operator|quarantined).*"),
	},
	groupConfig{
		name:    "network",
		version: regexp.MustCompile(versionPattern),
		sig:     regexp.MustCompile(`.*\\[sig-network].*`),
		lanes:   regexp.MustCompile(".*(network|quarantined).*"),
	},
	groupConfig{
		name:    "ssp",
		version: regexp.MustCompile(versionPattern),
		sig:     regexp.MustCompile(".*"),
		lanes:   regexp.MustCompile(".*ssp.*"),
	},
}

func runFilter(input *map[string]map[string]int, groupConfigs groupConfigs) map[string]map[string][]string {
	if groupConfigs == nil {
		groupConfigs = defaultGroupConfigs
	}
	result := map[string]map[string][]string{}
	for testName, testLanesToExecutions := range *input {
		groupsToVersions := map[string]map[string]struct{}{}
		for _, currentGroupConfig := range groupConfigs {
			if !currentGroupConfig.sig.MatchString(testName) {
				continue
			}

			type testState struct {
				run         bool
				unsupported bool
			}
			testStatePerVersion := map[string]*testState{}

			for testLane, testExecution := range testLanesToExecutions {
				if !currentGroupConfig.lanes.MatchString(testLane) || !currentGroupConfig.version.MatchString(testLane) {
					continue
				}
				version := currentGroupConfig.version.FindStringSubmatch(testLane)[1]
				if _, exists := testStatePerVersion[version]; !exists {
					testStatePerVersion[version] = &testState{}
				}
				if testExecution == testreport.TestExecution_Run {
					testStatePerVersion[version].run = true
				}
				if testExecution == testreport.TestExecution_Unsupported {
					testStatePerVersion[version].unsupported = true
				}
			}

			for version, testStateForVersion := range testStatePerVersion {
				if testStateForVersion.run || testStateForVersion.unsupported {
					continue
				}
				if _, exists := groupsToVersions[currentGroupConfig.name]; !exists {
					groupsToVersions[currentGroupConfig.name] = map[string]struct{}{}
				}
				if _, exists := groupsToVersions[currentGroupConfig.name][version]; !exists {
					groupsToVersions[currentGroupConfig.name][version] = struct{}{}
				}
			}
		}
		if len(groupsToVersions) == 0 {
			continue
		}
		for group, versions := range groupsToVersions {
			if len(versions) == 0 {
				continue
			}
			if _, exists := result[group]; !exists {
				result[group] = map[string][]string{}
			}
			for version := range versions {
				if _, exists := result[group][version]; !exists {
					result[group][version] = []string{testName}
				} else {
					result[group][version] = append(result[group][version], testName)
				}
			}
		}
	}
	for _, versions := range result {
		for _, testNames := range versions {
			sort.Strings(testNames)
		}
	}
	return result
}

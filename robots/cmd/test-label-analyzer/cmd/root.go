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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"kubevirt.io/project-infra/pkg/git"
	testlabelanalyzer "kubevirt.io/project-infra/pkg/test-label-analyzer"
	test_report "kubevirt.io/project-infra/pkg/test-report"

	"github.com/spf13/cobra"
	"kubevirt.io/project-infra/robots/cmd/test-label-analyzer/cmd/filter"
)

// ConfigOptions contains the set of options that the stats command provides
//
// one of ConfigFile or ConfigName is required
type ConfigOptions struct {

	// ConfigFile is the path to the configuration file that resembles the test_label_analyzer.Config
	ConfigFile string

	// ConfigName is the name of the default configuration that resembles the test_label_analyzer.Config
	ConfigName string

	// FilterTestNamesFile holds the file path to a filter file like quarantined_tests.json
	FilterTestNamesFile string

	// ginkgoOutlinePaths holds the paths to the files that contain the test outlines to analyze
	ginkgoOutlinePaths []string

	// testFilePath is the path to the files that contain the test code
	testFilePath string

	// remoteURL is the absolute path to the test files containing the test code with the analyzed state, most likely
	// containing a commit id defining the state of the observed outlines
	// if the url contains a pattern "%s" the tool will replace that with the latest commit id it finds
	remoteURL string

	// testNameLabelRE is the regular expression for an on the fly created configuration of test names to match against
	testNameLabelRE string

	// outputHTML defines whether HTML should be generated, default is JSON
	outputHTML bool

	// outputGCSURL defines the target where to put the generated output
	outputGCSURL string
}

// validate checks the configuration options for validity and returns an error describing the first error encountered
func (s *ConfigOptions) validate() error {
	if s.testNameLabelRE == "" {
		if s.ConfigFile == "" && s.ConfigName == "" && s.FilterTestNamesFile == "" {
			return fmt.Errorf("one of ConfigFile or ConfigName or FilterTestNamesFile is required")
		}
	}
	if _, exists := configNamesToConfigs[s.ConfigName]; s.ConfigName != "" && !exists {
		return fmt.Errorf("ConfigName %s is invalid", s.ConfigName)
	}
	if s.ConfigFile != "" {
		stat, err := os.Stat(s.ConfigFile)
		if os.IsNotExist(err) {
			return fmt.Errorf("config-file not set correctly, %q is not a file, %w", s.ConfigFile, err)
		}
		if stat.IsDir() {
			return fmt.Errorf("config-file not set correctly, %q is not a file", s.ConfigFile)
		}
	}
	for _, ginkgoOutlinePath := range s.ginkgoOutlinePaths {
		stat, err := os.Stat(ginkgoOutlinePath)
		if os.IsNotExist(err) {
			return fmt.Errorf("test-outline-filepath not set correctly, %q is not a file, %w", s.ginkgoOutlinePaths, err)
		}
		if stat.IsDir() {
			return fmt.Errorf("test-outline-filepath not set correctly, %q is not a file", s.ginkgoOutlinePaths)
		}
	}
	if s.testFilePath != "" {
		stat, err := os.Stat(s.testFilePath)
		if os.IsNotExist(err) {
			return fmt.Errorf("test-file-path not set correctly, %q is not a directory, %w", s.testFilePath, err)
		}
		if !stat.IsDir() {
			return fmt.Errorf("test-file-path not set correctly, %q is not a directory", s.testFilePath)
		}
		if s.remoteURL == "" {
			return fmt.Errorf("remote-url is required together with test-file-path")
		} else {
			if strings.Contains(s.remoteURL, "%s") {
				// fetch commit id and replace pattern with it
				// cd ../kubevirt && git --no-pager log -1 --format=%H | tr -d '\n'
				command := exec.Command("git", "--no-pager", "log", "-1", "--format=%H")
				command.Dir = s.testFilePath
				output, err := command.CombinedOutput()
				if err != nil {
					return err
				}
				s.remoteURL = fmt.Sprintf(s.remoteURL, strings.ReplaceAll(string(output), "\n", ""))
			}
		}
		if s.outputGCSURL != "" {
			gcsURLPattern := regexp.MustCompile("^gs://.*")
			if !gcsURLPattern.MatchString(s.outputGCSURL) {
				return fmt.Errorf("outputGCSURL must match pattern")
			}
		}
	}
	return nil
}

// getConfig returns a configuration with which the matching tests are being retrieved or an error in case the configuration is wrong
func (s *ConfigOptions) getConfig() (*testlabelanalyzer.Config, error) {
	switch {

	case s.testNameLabelRE != "":
		return testlabelanalyzer.NewTestNameDefaultConfig(s.testNameLabelRE), nil

	case s.ConfigName != "":
		config, exists := configNamesToConfigs[s.ConfigName]
		if !exists {
			return nil, fmt.Errorf("config %q does not exist", s.ConfigName)
		}
		return config, nil

	case s.ConfigFile != "":
		config, err := unmarshallConfigFile(s)
		return config, err

	case s.FilterTestNamesFile != "":
		filterTestRecords, err := unmarshallFilterTestRecords(s)
		if err != nil {
			return nil, err
		}
		blameLines, err := git.GetBlameLinesForFile(s.FilterTestNamesFile)
		if err != nil {
			return nil, err
		}
		return generateConfigWithCategoriesFromFilterTestRecords(filterTestRecords, blameLines), nil

	default:
		return nil, fmt.Errorf("no configuration found")
	}
}

func generateConfigWithCategoriesFromFilterTestRecords(filterTestRecords []test_report.FilterTestRecord, blameLines []*git.BlameLine) *testlabelanalyzer.Config {
	config := &testlabelanalyzer.Config{}
	blameIndex := 0
	for _, filterTestRecord := range filterTestRecords {
		quotedId := regexp.QuoteMeta(filterTestRecord.Id)
		testNameDefaultConfig := &testlabelanalyzer.LabelCategory{
			Name:            filterTestRecord.Reason,
			TestNameLabelRE: testlabelanalyzer.NewRegexp(quotedId),
			GinkgoLabelRE:   nil,
		}
		matchIdRegex := regexp.MustCompile(fmt.Sprintf(`"id":\s*"%s"`, quotedId))
		for index := blameIndex; index < len(blameLines); index++ {
			if !matchIdRegex.MatchString(blameLines[index].Line) {
				continue
			}
			testNameDefaultConfig.BlameLine = blameLines[index]
			blameIndex = index + 1
			break
		}
		config.Categories = append(config.Categories, testNameDefaultConfig)
	}
	return config
}

func unmarshallFilterTestRecords(s *ConfigOptions) ([]test_report.FilterTestRecord, error) {
	file, err := os.ReadFile(s.FilterTestNamesFile)
	if err != nil {
		return nil, err
	}
	var filterTestRecords []test_report.FilterTestRecord
	err = json.Unmarshal(file, &filterTestRecords)
	if err != nil {
		return nil, err
	}
	return filterTestRecords, nil
}

func unmarshallConfigFile(s *ConfigOptions) (*testlabelanalyzer.Config, error) {
	file, err := os.ReadFile(s.ConfigFile)
	if err != nil {
		return nil, err
	}
	var config *testlabelanalyzer.Config
	err = json.Unmarshal(file, &config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

var rootConfigOpts = ConfigOptions{}

var configNamesToConfigs = map[string]*testlabelanalyzer.Config{
	"quarantine": testlabelanalyzer.NewQuarantineDefaultConfig(),
}

const shortRootDescription = "Collects a set of tools for generating statistics and filter strings over sets of Ginkgo tests"

var RootCmd = &cobra.Command{
	Use:   "test-label-analyzer",
	Short: shortRootDescription,
	Long: shortRootDescription + `

Supports predefined configuration profiles and custom configurations to define which sets of tests should be targeted.`,
}

func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVar(&rootConfigOpts.ConfigFile, "config-file", "", "config file defining categories of tests")
	configNames := []string{}
	for configName := range configNamesToConfigs {
		configNames = append(configNames, configName)
	}
	RootCmd.PersistentFlags().StringVar(&rootConfigOpts.ConfigName, "config-name", "", fmt.Sprintf("config name defining categories of tests (possible values: %v)", configNames))
	RootCmd.PersistentFlags().StringArrayVar(&rootConfigOpts.ginkgoOutlinePaths, "test-outline-filepath", nil, "path to test outline file to be analyzed")
	RootCmd.PersistentFlags().StringVar(&rootConfigOpts.FilterTestNamesFile, "filter-test-names-file", "", "file path to filter file like quarantined_tests.json or dont_run_tests.json")
	RootCmd.PersistentFlags().StringVar(&rootConfigOpts.testFilePath, "test-file-path", "", "path containing tests to be analyzed")
	RootCmd.PersistentFlags().StringVar(&rootConfigOpts.remoteURL, "remote-url", "", "remote path to tests to be analyzed")
	RootCmd.PersistentFlags().StringVar(&rootConfigOpts.testNameLabelRE, "test-name-label-re", "", "regular expression for test names to match against")
	RootCmd.PersistentFlags().BoolVar(&rootConfigOpts.outputHTML, "output-html", false, "defines whether HTML output should be generated, default is JSON")
	RootCmd.PersistentFlags().StringVar(&rootConfigOpts.outputGCSURL, "output-gcs-url", "", "defines where to put the output (optional)")

	RootCmd.AddCommand(filter.Command())
}

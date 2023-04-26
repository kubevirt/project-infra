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
	"github.com/spf13/cobra"
	testlabelanalyzer "kubevirt.io/project-infra/robots/pkg/test-label-analyzer"
	"os"
)

// configOptions contains the set of options that the stats command provides
//
// one of configFile or configName is required
type configOptions struct {

	// configFile is the path to the configuration file that resembles the test_label_analyzer.Config
	configFile string

	// configName is the name of the default configuration that resembles the test_label_analyzer.Config
	configName string

	// ginkgoOutlinePaths holds the paths to the files that contain the test outlines to analyze
	ginkgoOutlinePaths []string

	// testFilePath is the path to the files that contain the test code
	testFilePath string

	// remoteURL is the absolute path to the test files containing the test code with the analyzed state, most likely
	// containing a commit id defining the state of the observed outlines
	remoteURL string

	// testNameLabelRE is the regular expression for an on the fly created configuration of test names to match against
	testNameLabelRE string

	// outputHTML defines whether HTML should be generated, default is JSON
	outputHTML bool
}

// validate checks the configuration options for validity and returns an error describing the first error encountered
func (s *configOptions) validate() error {
	if s.testNameLabelRE == "" {
		if s.configFile == "" && s.configName == "" || s.configFile != "" && s.configName != "" {
			return fmt.Errorf("one of configFile or configName is required")
		}
	}
	if _, exists := configNamesToConfigs[s.configName]; s.configName != "" && !exists {
		return fmt.Errorf("configName %s is invalid", s.configName)
	}
	if s.configFile != "" {
		stat, err := os.Stat(s.configFile)
		if os.IsNotExist(err) {
			return fmt.Errorf("config-file not set correctly, %q is not a file, %v", s.configFile, err)
		}
		if stat.IsDir() {
			return fmt.Errorf("config-file not set correctly, %q is not a file", s.configFile)
		}
	}
	for _, ginkgoOutlinePath := range s.ginkgoOutlinePaths {
		stat, err := os.Stat(ginkgoOutlinePath)
		if os.IsNotExist(err) {
			return fmt.Errorf("test-outline-filepath not set correctly, %q is not a file, %v", s.ginkgoOutlinePaths, err)
		}
		if stat.IsDir() {
			return fmt.Errorf("test-outline-filepath not set correctly, %q is not a file", s.ginkgoOutlinePaths)
		}
	}
	if s.testFilePath != "" {
		stat, err := os.Stat(s.testFilePath)
		if os.IsNotExist(err) {
			return fmt.Errorf("test-file-path not set correctly, %q is not a directory, %v", s.testFilePath, err)
		}
		if !stat.IsDir() {
			return fmt.Errorf("test-file-path not set correctly, %q is not a directory", s.testFilePath)
		}
		if s.remoteURL == "" {
			return fmt.Errorf("remote-url is required together with test-file-path")
		}
	}
	return nil
}

// getConfig returns a configuration with which the matching tests are being retrieved or an error in case the configuration is wrong
func (s *configOptions) getConfig() (*testlabelanalyzer.Config, error) {
	if s.testNameLabelRE != "" {
		return testlabelanalyzer.NewTestNameDefaultConfig(s.testNameLabelRE), nil
	}
	if s.configName != "" {
		return configNamesToConfigs[s.configName], nil
	}
	if s.configFile != "" {
		file, err := os.ReadFile(s.configFile)
		if err != nil {
			return nil, err
		}
		var config *testlabelanalyzer.Config
		err = json.Unmarshal(file, &config)
		return config, err
	}
	return nil, fmt.Errorf("no configuration found!")
}

var rootConfigOpts = configOptions{}

var configNamesToConfigs = map[string]*testlabelanalyzer.Config{
	"quarantine": testlabelanalyzer.NewQuarantineDefaultConfig(),
}

const shortRootDescription = "Collects a set of tools for generating statistics and filter strings over sets of Ginkgo tests"

var rootCmd = &cobra.Command{
	Use:   "test-label-analyzer",
	Short: shortRootDescription,
	Long: shortRootDescription + `

Supports predefined configuration profiles and custom configurations to define which sets of tests should be targeted.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&rootConfigOpts.configFile, "config-file", "", "config file defining categories of tests")
	configNames := []string{}
	for configName := range configNamesToConfigs {
		configNames = append(configNames, configName)
	}
	rootCmd.PersistentFlags().StringVar(&rootConfigOpts.configName, "config-name", "", fmt.Sprintf("config name defining categories of tests (possible values: %v)", configNames))
	rootCmd.PersistentFlags().StringArrayVar(&rootConfigOpts.ginkgoOutlinePaths, "test-outline-filepath", nil, "path to test outline file to be analyzed")
	rootCmd.PersistentFlags().StringVar(&rootConfigOpts.testFilePath, "test-file-path", "", "path containing tests to be analyzed")
	rootCmd.PersistentFlags().StringVar(&rootConfigOpts.remoteURL, "remote-url", "", "remote path to tests to be analyzed")
	rootCmd.PersistentFlags().StringVar(&rootConfigOpts.testNameLabelRE, "test-name-label-re", "", "regular expression for test names to match against")
	rootCmd.PersistentFlags().BoolVar(&rootConfigOpts.outputHTML, "output-html", false, "defines whether HTML output should be generated, default is JSON")
}

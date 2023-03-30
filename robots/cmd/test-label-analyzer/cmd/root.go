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
	"github.com/spf13/cobra"
	test_label_analyzer "kubevirt.io/project-infra/robots/pkg/test-label-analyzer"
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

	// ginkgoOutlinePathes is the pathes to the files that contain the test outlines to analyze
	ginkgoOutlinePathes []string
}

func (s *configOptions) validate() error {
	if s.configFile == "" && s.configName == "" || s.configFile != "" && s.configName != "" {
		return fmt.Errorf("one of configFile or configName is required")
	}
	for _, ginkgoOutlinePath := range s.ginkgoOutlinePathes {
		stat, err := os.Stat(ginkgoOutlinePath)
		if os.IsNotExist(err) {
			return fmt.Errorf("test-outline-filepath not set correctly, %q is not a file, %v", s.ginkgoOutlinePathes, err)
		}
		if stat.IsDir() {
			return fmt.Errorf("test-outline-filepath not set correctly, %q is not a file", s.ginkgoOutlinePathes)
		}
	}
	return nil
}

var configOpts = configOptions{}

var configNamesToConfigs = map[string]*test_label_analyzer.Config{
	"quarantine": test_label_analyzer.NewQuarantineDefaultConfig(),
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "test-label-analyzer",
	Short: "blah",
	Long:  `TODO`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configOpts.configFile, "config-file", "", "config file defining categories of tests")
	configNames := []string{}
	for configName := range configNamesToConfigs {
		configNames = append(configNames, configName)
	}
	rootCmd.PersistentFlags().StringVar(&configOpts.configName, "config-name", "", fmt.Sprintf("config name defining categories of tests (possible values: %v)", configNames))
	rootCmd.PersistentFlags().StringArrayVar(&configOpts.ginkgoOutlinePathes, "test-outline-filepath", nil, "path containing tests to be analyzed")
}

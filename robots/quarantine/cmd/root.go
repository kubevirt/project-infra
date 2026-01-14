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
 * Copyright the KubeVirt Authors.
 *
 */

package cmd

import (
	"fmt"
	"os"

	"github.com/onsi/ginkgo/v2/types"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"kubevirt.io/project-infra/pkg/ginkgo"
)

var rootCmd = &cobra.Command{
	Use: "quarantine",
}

var quarantineOpts quarantineOptions

func init() {
	rootCmd.PersistentFlags().StringVar(&quarantineOpts.testSourcePath, "test-source-path", "../kubevirt/tests/", "the path to the Test source")

	rootCmd.AddCommand(mostFlakyTestsReportCmd)
	rootCmd.AddCommand(quarantineTestCmd)
	rootCmd.AddCommand(infoTestCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		logrus.WithError(err).Error("failed to run command")
		os.Exit(1)
	}
}

func getDryRunSpecReport(quarantineOpts *quarantineOptions) (*types.SpecReport, error) {
	reports, _, err := ginkgo.DryRun(quarantineOpts.testSourcePath)
	if err != nil {
		return nil, fmt.Errorf("could not locate test named %q: %w", quarantineOpts.testName, err)
	}
	if reports == nil {
		return nil, fmt.Errorf("could not generate test file reports for test named %q", quarantineOpts.testName)
	}
	matchingSpecReport := ginkgo.GetSpecReportByTestName(reports, quarantineOpts.testName)
	if matchingSpecReport == nil {
		return nil, fmt.Errorf("could not locate test named %q ", quarantineOpts.testName)
	}
	return matchingSpecReport, nil
}

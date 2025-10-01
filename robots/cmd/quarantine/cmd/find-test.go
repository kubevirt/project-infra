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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"kubevirt.io/project-infra/robots/pkg/ginkgo"
	"os"
	"path/filepath"
)

const (
	shortFindTest = `Find a test given by test name`
)

var findTestCmd = &cobra.Command{
	Use:   "find",
	Short: shortFindTest,
	Long: shortFindTest + `, i.o.w.
TODO`,
	RunE: FindTest,
}

func init() {
	findTestCmd.PersistentFlags().StringVar(&quarantineOpts.testName, "test-name", "", "the name of the Test to quarantine")
}

func FindTest(_ *cobra.Command, _ []string) error {

	reports, output, err := ginkgo.DryRun(quarantineOpts.testSourcePath)
	log.Info(string(output))
	defer os.Remove(filepath.Join(quarantineOpts.testSourcePath, "junit.functest.xml"))
	if err != nil {
		return fmt.Errorf("could not find test file for %q by name: %w", quarantineOpts.testName, err)
	}
	if reports == nil {
		return fmt.Errorf("could not generate test file reports for %q", quarantineOpts.testName)
	}
	matchingSpecReport := ginkgo.GetMatchingSpecReport(reports, quarantineOpts.testName)
	if matchingSpecReport == nil {
		return fmt.Errorf("could not find test file for %q by name", quarantineOpts.testName)
	}

	log.Infof("test %q located in file %q on line %d", matchingSpecReport.FullText(), matchingSpecReport.LeafNodeLocation.FileName, matchingSpecReport.LeafNodeLocation.LineNumber)

	return nil
}

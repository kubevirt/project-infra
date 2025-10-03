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
	short = `Quarantine a test given by test name`
)

var quarantineTestCmd = &cobra.Command{
	Use:   "test",
	Short: short,
	Long: short + `, i.o.w.
modify a test by attaching the label and decorator
as defined in KubeVirt docs. This prepares it to be recognized by CI automation
that the test is to be excluded from e2e presubmit lane runs and only run in
periodic tests.

More information about quarantining e2e tests in KubeVirt can be found here:
https://github.com/kubevirt/kubevirt/blob/main/docs/quarantine.md#quarantine-pr`,
	RunE: QuarantineTest,
}

func init() {
	quarantineTestCmd.PersistentFlags().StringVar(&quarantineOpts.testName, "test-name", "", "the name of the Test to quarantine")
}

func QuarantineTest(cmd *cobra.Command, args []string) error {

	reports, _, err := ginkgo.DryRun(quarantineOpts.testSourcePath)
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

	err = ginkgo.QuarantineTestInFile(matchingSpecReport)
	if err != nil {
		return err
	}

	log.Infof("test %q quarantined in file %q", matchingSpecReport.FullText(), matchingSpecReport.LeafNodeLocation.FileName)

	return nil
}

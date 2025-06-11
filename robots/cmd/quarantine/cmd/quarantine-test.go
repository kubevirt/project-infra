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
)

var quarantineTestCmd = &cobra.Command{
	Use:  "test",
	RunE: QuarantineTest,
}

func init() {
	quarantineTestCmd.PersistentFlags().StringVar(&quarantineOpts.testName, "test-name", "", "the name of the test to quarantine")
}

func QuarantineTest(cmd *cobra.Command, args []string) error {

	descriptor, _, err := ginkgo.FindFileAndDescriptor(quarantineOpts.testSourcePath, quarantineOpts.testName)
	if err != nil {
		return fmt.Errorf("could not find file or descriptor for test %q: %w", &quarantineOpts.testName, err)
	}

	log.Infof("preparing to quarantine test %q with data %+v", descriptor.OutlineNode().Text, descriptor.OutlineNode())

	return nil
}

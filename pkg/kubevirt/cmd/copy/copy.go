/*
 * Copyright 2021 The KubeVirt Authors.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package copy

import (
	"fmt"

	"github.com/spf13/cobra"
)

var copyCommand = &cobra.Command{
	Use:   "copy",
	Short: "kubevirt copy copies job definitions in project-infra for kubevirt/kubevirt repo from existing provider jobs",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString())
	},
}

func init() {
	copyCommand.AddCommand(CopyJobsCommand())
}

func CopyCommand() *cobra.Command {
	return copyCommand
}

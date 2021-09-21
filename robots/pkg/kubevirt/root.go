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

package kubevirt

import (
	"fmt"
	"github.com/spf13/cobra"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/copy"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/flags"
	"kubevirt.io/project-infra/robots/pkg/kubevirt/require"
	"os"
)

var rootCmd = &cobra.Command{
	Use: "kubevirt",
	Short: "kubevirt alters job definitions in project-infra for kubevirt/kubevirt repo",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString())
	},
}

func init() {
	flags.AddPersistentFlags(rootCmd)

	rootCmd.AddCommand(copy.NewCopyCommand())
	rootCmd.AddCommand(require.NewRequireCommand())
}

func Execute()  {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

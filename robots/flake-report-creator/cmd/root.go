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

package cmd

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"os"
)

type GlobalOptions struct {
	outputFile string
	overwrite  bool
}

func (o *GlobalOptions) Validate() error {
	if o.outputFile == "" {
		outputFile, err := os.CreateTemp("", "flakefinder-*.html")
		if err != nil {
			return fmt.Errorf("failed to write report: %v", err)
		}
		o.outputFile = outputFile.Name()
	} else {
		stat, err := os.Stat(o.outputFile)
		if err != nil && errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if stat.IsDir() {
			return fmt.Errorf("failed to write report, file %s is a directory", o.outputFile)
		}
		if err == nil && !o.overwrite {
			return fmt.Errorf("failed to write report, file %s exists", o.outputFile)
		}
	}
	return nil
}

var (
	rootCmd = &cobra.Command{
		Use:   "flake-report-creator",
		Short: "flake-report-creator creates reports from junit artifacts of kubevirt ci builds",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString())
		},
	}
	globalOpts = GlobalOptions{}
)

func init() {
	rootCmd.PersistentFlags().StringVar(&globalOpts.outputFile, "outputFile", "", "Path to output file, if not given, a temporary file will be used")
	rootCmd.PersistentFlags().BoolVar(&globalOpts.overwrite, "overwrite", false, "Whether to overwrite output file")

	rootCmd.AddCommand(JenkinsCommand())
	rootCmd.AddCommand(ProwCommand())
}

func Execute() error {
	return rootCmd.Execute()
}

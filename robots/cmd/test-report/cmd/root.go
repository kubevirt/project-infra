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
 * Copyright 2022 Red Hat, Inc.
 *
 */

package cmd

import (
	_ "embed"
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"kubevirt.io/project-infra/robots/cmd/test-report/cmd/dequarantine"
	"kubevirt.io/project-infra/robots/cmd/test-report/cmd/execution"
	"kubevirt.io/project-infra/robots/cmd/test-report/cmd/filter"
)

var rootCmd *cobra.Command

func init() {
	rootCmd = &cobra.Command{
		Use:   "test-report",
		Short: "test-report creates reports about test executions",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprint(cmd.OutOrStderr(), cmd.UsageString())
		},
	}

	rootCmd.PersistentFlags().Uint32Var(&rootOpts.logLevel, "log-level", 4, "level for logging")

	log.SetLevel(log.Level(rootOpts.logLevel))
	log.SetFormatter(&log.JSONFormatter{})
	logger := log.StandardLogger().WithField("robot", "test-report")

	rootCmd.AddCommand(execution.ExecutionCmd(logger))
	rootCmd.AddCommand(dequarantine.DequarantineCmd(logger))
	rootCmd.AddCommand(filter.FilterCmd())
}

func Execute() error {
	if err := rootOpts.Validate(); err != nil {
		return err
	}
	return rootCmd.Execute()
}

type rootOptions struct {
	logLevel uint32
}

func (r rootOptions) Validate() error {
	if log.Level(r.logLevel) < log.PanicLevel || log.Level(r.logLevel) > log.TraceLevel {
		return fmt.Errorf("LogLevel %d out of bounds", r.logLevel)
	}
	return nil
}

var rootOpts = rootOptions{}

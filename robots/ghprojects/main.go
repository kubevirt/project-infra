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

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"kubevirt.io/project-infra/pkg/ghprojects/client"
	"kubevirt.io/project-infra/pkg/ghprojects/config"
	"kubevirt.io/project-infra/pkg/ghprojects/reconcile"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ghprojects",
		Short: "Manage GitHub Projects V2 declaratively from YAML configuration",
	}

	rootCmd.AddCommand(syncCmd())
	rootCmd.AddCommand(dumpCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func syncCmd() *cobra.Command {
	var (
		configPath  string
		tokenPath   string
		confirm     bool
		fixAll      bool
		fixProjects bool
		fixFields   bool
		fixViews    bool
		debug       bool
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize GitHub Projects V2 to match the YAML configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if debug {
				logrus.SetLevel(logrus.DebugLevel)
			}

			token, err := readToken(tokenPath)
			if err != nil {
				return err
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return err
			}

			c := client.New(token)

			opts := reconcile.Options{
				Confirm:     confirm,
				FixProjects: fixAll || fixProjects,
				FixFields:   fixAll || fixFields,
				FixViews:    fixAll || fixViews,
			}

			if !opts.FixProjects && !opts.FixFields && !opts.FixViews {
				opts.FixProjects = true
				opts.FixFields = true
				opts.FixViews = true
			}

			if !confirm {
				logrus.Info("running in dry-run mode, use --confirm to apply changes")
			}

			return reconcile.Reconcile(context.Background(), c, cfg, opts)
		},
	}

	cmd.Flags().StringVar(&configPath, "config-path", "", "Path to the projects YAML configuration file (required)")
	cmd.Flags().StringVar(&tokenPath, "github-token-path", "/etc/github/oauth", "Path to the file containing the GitHub OAuth token")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "Apply changes (default is dry-run)")
	cmd.Flags().BoolVar(&fixAll, "fix-all", false, "Fix all resource types (projects, fields, views)")
	cmd.Flags().BoolVar(&fixProjects, "fix-projects", false, "Create/update projects")
	cmd.Flags().BoolVar(&fixFields, "fix-fields", false, "Create/update custom fields")
	cmd.Flags().BoolVar(&fixViews, "fix-views", false, "Create/update views")
	cmd.Flags().BoolVar(&debug, "v", false, "Enable debug logging")
	if err := cmd.MarkFlagRequired("config-path"); err != nil {
		logrus.WithError(err).Fatal("failed to mark config-path flag as required")
	}

	return cmd
}

func dumpCmd() *cobra.Command {
	var (
		tokenPath string
		org       string
		debug     bool
	)

	cmd := &cobra.Command{
		Use:   "dump",
		Short: "Export the current GitHub Projects V2 state as YAML",
		RunE: func(cmd *cobra.Command, args []string) error {
			if debug {
				logrus.SetLevel(logrus.DebugLevel)
			}

			token, err := readToken(tokenPath)
			if err != nil {
				return err
			}

			c := client.New(token)

			cfg, err := config.Dump(context.Background(), c, org)
			if err != nil {
				return err
			}

			out, err := yaml.Marshal(cfg)
			if err != nil {
				return fmt.Errorf("marshaling config: %w", err)
			}

			fmt.Print(string(out))
			return nil
		},
	}

	cmd.Flags().StringVar(&tokenPath, "github-token-path", "/etc/github/oauth", "Path to the file containing the GitHub OAuth token")
	cmd.Flags().StringVar(&org, "org", "", "GitHub organization to dump projects from (required)")
	cmd.Flags().BoolVar(&debug, "v", false, "Enable debug logging")
	if err := cmd.MarkFlagRequired("org"); err != nil {
		logrus.WithError(err).Fatal("failed to mark org flag as required")
	}

	return cmd
}

func readToken(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading token from %s: %w", path, err)
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("token file %s is empty", path)
	}
	return token, nil
}

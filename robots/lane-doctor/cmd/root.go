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
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v32/github"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

var (
	repo      string
	tokenPath string
	dryRun    bool
)

var rootCmd = &cobra.Command{
	Use:   "lane-doctor",
	Short: "Diagnose and remediate stuck Prow lane statuses on open PRs",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&repo, "repo", "kubevirt/kubevirt", "GitHub repository in owner/repo format")
	rootCmd.PersistentFlags().StringVar(&tokenPath, "token-path", "", "path to GitHub token file (falls back to GITHUB_TOKEN env var)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print actions without executing them")

	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(prioritizeCmd)
	rootCmd.AddCommand(triggerCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func parseRepo() (owner, repoName string, err error) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repo format %q, expected owner/repo", repo)
	}
	return parts[0], parts[1], nil
}

func newGitHubClient(ctx context.Context) (*github.Client, error) {
	token, err := resolveToken()
	if err != nil {
		return nil, err
	}
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	return github.NewClient(oauth2.NewClient(ctx, ts)), nil
}

func resolveToken() (string, error) {
	if tokenPath != "" {
		data, err := os.ReadFile(tokenPath)
		if err != nil {
			return "", fmt.Errorf("reading token file: %w", err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		return t, nil
	}
	return "", fmt.Errorf("no GitHub token: set --token-path or GITHUB_TOKEN env var")
}

func writeOutput(data []byte, outputPath string) error {
	if outputPath == "" {
		_, err := os.Stdout.Write(data)
		return err
	}
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}
	logrus.WithField("path", outputPath).Info("wrote output file")
	return nil
}

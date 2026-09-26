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
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/prow/pkg/config/secret"
	github "sigs.k8s.io/prow/pkg/github"
)

type ghClient interface {
	GetBranchProtection(org, repo, branch string) (*github.BranchProtection, error)
	GetPullRequests(org, repo string) ([]github.PullRequest, error)
	GetCombinedStatus(org, repo, ref string) (*github.CombinedStatus, error)
	ListStatuses(org, repo, ref string) ([]github.Status, error)
	CreateComment(org, repo string, number int, comment string) error
}

var (
	repo      string
	tokenPath string
	dryRun    bool
	endpoint  string
)

var rootCmd = &cobra.Command{
	Use:   "lane-doctor",
	Short: "Diagnose and remediate stuck Prow lane statuses on open PRs",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&repo, "repo", "kubevirt/kubevirt", "GitHub repository in owner/repo format")
	rootCmd.PersistentFlags().StringVar(&tokenPath, "token-path", "", "path to GitHub token file (falls back to GITHUB_TOKEN env var)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print actions without executing them")
	rootCmd.PersistentFlags().StringVar(&endpoint, "endpoint", github.DefaultAPIEndpoint, "GitHub API endpoint")

	rootCmd.AddCommand(scanCmd)
	rootCmd.AddCommand(prioritizeCmd)
	rootCmd.AddCommand(triggerCmd)
}

// Execute runs the root command.
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

func newGitHubClient() (ghClient, error) {
	path, err := resolveTokenPath()
	if err != nil {
		return nil, err
	}
	if err := secret.Add(path); err != nil {
		return nil, fmt.Errorf("starting secrets agent: %w", err)
	}
	if dryRun {
		return github.NewDryRunClient(secret.GetTokenGenerator(path), secret.Censor, "", endpoint)
	}
	return github.NewClient(secret.GetTokenGenerator(path), secret.Censor, "", endpoint)
}

func resolveTokenPath() (string, error) {
	if tokenPath != "" {
		return tokenPath, nil
	}
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		f, err := os.CreateTemp("", "lane-doctor-token-*")
		if err != nil {
			return "", fmt.Errorf("creating temp token file: %w", err)
		}
		if _, err := f.WriteString(t); err != nil {
			f.Close()
			return "", fmt.Errorf("writing temp token file: %w", err)
		}
		if err := f.Close(); err != nil {
			return "", fmt.Errorf("closing temp token file: %w", err)
		}
		return f.Name(), nil
	}
	return "", fmt.Errorf("no GitHub token: set --token-path or GITHUB_TOKEN env var")
}

func writeOutput(data []byte, outputPath string) error {
	if outputPath == "" {
		if _, err := os.Stdout.Write(data); err != nil {
			return fmt.Errorf("writing to stdout: %w", err)
		}
		return nil
	}
	if err := os.WriteFile(outputPath, data, 0644); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}
	logrus.WithField("path", outputPath).Info("wrote output file")
	return nil
}

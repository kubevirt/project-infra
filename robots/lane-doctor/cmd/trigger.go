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
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

var (
	triggerInput     string
	triggerBatchSize int
	triggerGroup     string
	triggerBatchWait time.Duration
	triggerYes       bool
)

var triggerCmd = &cobra.Command{
	Use:   "trigger",
	Short: "Trigger the stuck lane on PRs in batches",
	RunE:  runTrigger,
	PreRunE: func(cmd *cobra.Command, _ []string) error {
		if triggerYes && !cmd.Flags().Changed("batch-wait") {
			return fmt.Errorf("--yes requires --batch-wait to be explicitly set")
		}
		if triggerBatchSize < 1 {
			return fmt.Errorf("--batch-size must be a positive integer, got %d", triggerBatchSize)
		}
		if triggerBatchWait < 0 {
			return fmt.Errorf("--batch-wait cannot be negative")
		}
		return nil
	},
}

func init() {
	triggerCmd.Flags().StringVarP(&triggerInput, "input", "i", "", "path to priority report YAML (required)")
	triggerCmd.Flags().IntVarP(&triggerBatchSize, "batch-size", "b", 10, "number of PRs to trigger per batch")
	triggerCmd.Flags().StringVarP(&triggerGroup, "group", "g", "", "trigger only a specific priority group (e.g. P1)")
	triggerCmd.Flags().DurationVar(&triggerBatchWait, "batch-wait", 4*time.Hour, "wait duration between batches")
	triggerCmd.Flags().BoolVarP(&triggerYes, "yes", "y", false, "auto-continue between batches (requires --batch-wait)")
	_ = triggerCmd.MarkFlagRequired("input")
}

func runTrigger(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	owner, repoName, err := parseRepo()
	if err != nil {
		return err
	}

	data, err := os.ReadFile(triggerInput)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	var report PriorityReport
	if err := yaml.Unmarshal(data, &report); err != nil {
		return fmt.Errorf("parsing priority report: %w", err)
	}

	prNumbers := collectPRNumbers(report.Groups, triggerGroup)
	if len(prNumbers) == 0 {
		logrus.Info("no PRs to trigger")
		return nil
	}

	batches := splitBatches(prNumbers, triggerBatchSize)
	logrus.WithFields(logrus.Fields{
		"total_prs":  len(prNumbers),
		"batches":    len(batches),
		"batch_size": triggerBatchSize,
	}).Info("triggering lane")

	var client ghClient
	if !dryRun {
		client, err = newGitHubClient()
		if err != nil {
			return err
		}
	}

	comment := fmt.Sprintf("/test %s", report.Lane)

	for i, batch := range batches {
		logrus.WithFields(logrus.Fields{
			"batch":    i + 1,
			"of":       len(batches),
			"pr_count": len(batch),
		}).Info("processing batch")

		for _, prNum := range batch {
			if dryRun {
				fmt.Printf("[dry-run] would comment on PR #%d: %s\n", prNum, comment)
				continue
			}
			if err := postComment(client, owner, repoName, prNum, comment); err != nil {
				logrus.WithError(err).WithField("pr", prNum).Error("failed to post comment")
				continue
			}
		}

		if i < len(batches)-1 {
			if err := waitBetweenBatches(ctx, i+1, len(batches)); err != nil {
				return err
			}
		}
	}

	logrus.Info("all batches processed")
	return nil
}

func collectPRNumbers(groups []PriorityGroup, filterGroup string) []int {
	var numbers []int
	for _, g := range groups {
		if filterGroup != "" && g.Name != filterGroup {
			continue
		}
		numbers = append(numbers, g.PRNumbers...)
	}
	return numbers
}

func splitBatches(items []int, size int) [][]int {
	var batches [][]int
	for i := 0; i < len(items); i += size {
		end := i + size
		if end > len(items) {
			end = len(items)
		}
		batches = append(batches, items[i:end])
	}
	return batches
}

func postComment(client ghClient, owner, repoName string, prNumber int, body string) error {
	if err := client.CreateComment(owner, repoName, prNumber, body); err != nil {
		return fmt.Errorf("commenting on PR #%d: %w", prNumber, err)
	}
	logrus.WithField("pr", prNumber).Info("triggered lane")
	return nil
}

func waitBetweenBatches(ctx context.Context, completedBatch, totalBatches int) error {
	if dryRun {
		fmt.Printf("[dry-run] would wait %s before batch %d/%d\n", triggerBatchWait, completedBatch+1, totalBatches)
		return nil
	}

	if triggerYes {
		logrus.WithFields(logrus.Fields{
			"wait":       triggerBatchWait,
			"next_batch": completedBatch + 1,
		}).Info("waiting before next batch")

		deadline := time.Now().Add(triggerBatchWait)
		timer := time.NewTimer(time.Until(deadline))
		defer timer.Stop()
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			case <-ticker.C:
				remaining := time.Until(deadline).Truncate(time.Second)
				logrus.WithField("remaining", remaining).Info("waiting for next batch")
			}
		}
	}

	fmt.Printf("\nBatch %d/%d complete. Press Enter to continue (or Ctrl+C to abort)... ", completedBatch, totalBatches)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}
	_ = strings.TrimSpace(scanner.Text())
	return nil
}

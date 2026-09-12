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
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

var (
	prioritizeInput  string
	prioritizeOutput string
)

var prioritizeCmd = &cobra.Command{
	Use:   "prioritize",
	Short: "Group stuck PRs into priority tiers based on merge-readiness labels",
	Long: `Group stuck PRs from a scan report into priority tiers:

  P1  lgtm + approved     — blocked from merging, highest priority
  P2  lgtm or approved    — close to merging
  P3  no merge labels     — under review
  P4  do-not-merge/hold   — on hold

Draft PRs and PRs labeled do-not-merge/work-in-progress are excluded.`,
	RunE: runPrioritize,
}

func init() {
	prioritizeCmd.Flags().StringVarP(&prioritizeInput, "input", "i", "", "path to scan report YAML (required)")
	prioritizeCmd.Flags().StringVarP(&prioritizeOutput, "output", "o", "", "output YAML file path (default: stdout)")
	_ = prioritizeCmd.MarkFlagRequired("input")
}

func runPrioritize(_ *cobra.Command, _ []string) error {
	data, err := os.ReadFile(prioritizeInput)
	if err != nil {
		return fmt.Errorf("reading input: %w", err)
	}

	var scan ScanReport
	if err := yaml.Unmarshal(data, &scan); err != nil {
		return fmt.Errorf("parsing scan report: %w", err)
	}

	groups := classify(scan.StuckPRs)

	report := PriorityReport{
		Lane:          scan.Lane,
		Repo:          scan.Repo,
		PrioritizedAt: time.Now().UTC().Format(time.RFC3339),
		Groups:        groups,
	}

	out, err := yaml.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshaling priority report: %w", err)
	}
	return writeOutput(out, prioritizeOutput)
}

func classify(prs []StuckPR) []PriorityGroup {
	var p1, p2, p3, p4 []int

	for _, pr := range prs {
		if pr.IsDraft || hasLabel(pr.Labels, "do-not-merge/work-in-progress") {
			continue
		}

		hold := hasLabel(pr.Labels, "do-not-merge/hold")
		lgtm := hasLabel(pr.Labels, "lgtm")
		approved := hasAnyApprovedLabel(pr.Labels)

		switch {
		case hold:
			p4 = append(p4, pr.Number)
		case lgtm && approved:
			p1 = append(p1, pr.Number)
		case lgtm || approved:
			p2 = append(p2, pr.Number)
		default:
			p3 = append(p3, pr.Number)
		}
	}

	sort.Ints(p1)
	sort.Ints(p2)
	sort.Ints(p3)
	sort.Ints(p4)

	var groups []PriorityGroup
	if len(p1) > 0 {
		groups = append(groups, PriorityGroup{Name: "P1", Description: "lgtm + approved — blocked from merging", PRNumbers: p1})
	}
	if len(p2) > 0 {
		groups = append(groups, PriorityGroup{Name: "P2", Description: "lgtm or approved — close to merging", PRNumbers: p2})
	}
	if len(p3) > 0 {
		groups = append(groups, PriorityGroup{Name: "P3", Description: "under review", PRNumbers: p3})
	}
	if len(p4) > 0 {
		groups = append(groups, PriorityGroup{Name: "P4", Description: "on hold", PRNumbers: p4})
	}
	return groups
}

func hasLabel(labels []string, target string) bool {
	for _, l := range labels {
		if l == target {
			return true
		}
	}
	return false
}

func hasAnyApprovedLabel(labels []string) bool {
	for _, l := range labels {
		if l == "approved" || strings.HasPrefix(l, "approved-") {
			return true
		}
	}
	return false
}

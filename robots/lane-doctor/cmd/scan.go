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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	github "sigs.k8s.io/prow/pkg/github"
	"sigs.k8s.io/yaml"
)

var (
	scanLane   string
	scanOutput string
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan open PRs for stuck or missing lane statuses",
	RunE:  runScan,
}

func init() {
	scanCmd.Flags().StringVar(&scanLane, "lane", "", "Prow job name or GitHub status context (they are usually identical)")
	scanCmd.Flags().StringVarP(&scanOutput, "output", "o", "", "output YAML file path (default: stdout)")
	_ = scanCmd.MarkFlagRequired("lane")
}

func runScan(cmd *cobra.Command, _ []string) error {
	owner, repoName, err := parseRepo()
	if err != nil {
		return err
	}

	client, err := newGitHubClient()
	if err != nil {
		return err
	}

	log := logrus.WithFields(logrus.Fields{"lane": scanLane, "repo": repo})

	if err := verifyRequiredCheck(client, owner, repoName, scanLane); err != nil {
		return err
	}
	log.Info("confirmed lane is a required status check")

	prs, err := listOpenPRs(client, owner, repoName)
	if err != nil {
		return err
	}
	log.WithField("count", len(prs)).Info("fetched open PRs")

	summary, stuckPRs := classifyPRs(client, owner, repoName, scanLane, prs)

	sort.Slice(stuckPRs, func(i, j int) bool {
		return stuckPRs[i].Number < stuckPRs[j].Number
	})

	report := ScanReport{
		Lane:      scanLane,
		Repo:      repo,
		ScannedAt: time.Now().UTC().Format(time.RFC3339),
		Summary:   summary,
		StuckPRs:  stuckPRs,
	}

	data, err := yaml.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshaling report: %w", err)
	}
	return writeOutput(data, scanOutput)
}

func verifyRequiredCheck(client ghClient, owner, repoName, lane string) error {
	protection, err := client.GetBranchProtection(owner, repoName, "main")
	if err != nil {
		return fmt.Errorf("fetching branch protection: %w", err)
	}
	if protection == nil || protection.RequiredStatusChecks == nil {
		return fmt.Errorf("no required status checks configured on main")
	}
	for _, check := range protection.RequiredStatusChecks.Contexts {
		if check == lane {
			return nil
		}
	}
	return fmt.Errorf("lane %q is not a required status check on main", lane)
}

func listOpenPRs(client ghClient, owner, repoName string) ([]github.PullRequest, error) {
	prs, err := client.GetPullRequests(owner, repoName)
	if err != nil {
		return nil, fmt.Errorf("listing PRs: %w", err)
	}
	return prs, nil
}

type prResult struct {
	category string // "stuck", "missing", "running", "success", "failed"
	pr       *StuckPR
}

func classifyPRs(client ghClient, owner, repoName, lane string, prs []github.PullRequest) (ScanSummary, []StuckPR) {
	const workers = 10
	sem := make(chan struct{}, workers)
	results := make([]prResult, len(prs))
	var wg sync.WaitGroup

	for i := range prs {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx] = classifyPR(client, owner, repoName, lane, &prs[idx])
		}(i)
	}
	wg.Wait()

	var summary ScanSummary
	var stuckPRs []StuckPR
	summary.Total = len(prs)

	for _, r := range results {
		switch r.category {
		case "stuck":
			summary.Stuck++
		case "missing":
			summary.Missing++
		case "running":
			summary.Running++
		case "success":
			summary.Success++
		case "failed":
			summary.Failed++
		}
		if r.pr != nil {
			stuckPRs = append(stuckPRs, *r.pr)
		}
	}
	return summary, stuckPRs
}

func classifyPR(client ghClient, owner, repoName, lane string, pr *github.PullRequest) prResult {
	log := logrus.WithField("pr", pr.Number)
	sha := pr.Head.SHA

	laneStatus, hasE2E, err := getLaneStatus(client, owner, repoName, sha, lane)
	if err != nil {
		log.WithError(err).Warn("failed to get combined status")
		return prResult{category: "failed"}
	}

	if laneStatus == nil {
		if !hasE2E {
			return prResult{category: "success"}
		}
		return prResult{category: "missing", pr: buildStuckPR(pr, "missing", "", false)}
	}

	switch laneStatus.State {
	case "success":
		return prResult{category: "success"}
	case "failure", "error":
		return prResult{category: "failed"}
	case "pending":
		hasURL, statusUpdatedAt := checkRawStatuses(client, owner, repoName, sha, lane)
		if hasURL {
			return prResult{category: "running"}
		}
		return prResult{category: "stuck", pr: buildStuckPR(pr, "pending", statusUpdatedAt, false)}
	default:
		return prResult{category: "failed"}
	}
}

func getLaneStatus(client ghClient, owner, repoName, sha, lane string) (laneStatus *github.Status, hasE2E bool, err error) {
	combined, err := client.GetCombinedStatus(owner, repoName, sha)
	if err != nil {
		return nil, false, err
	}
	for i, s := range combined.Statuses {
		if s.Context == lane {
			laneStatus = &combined.Statuses[i]
		}
		if strings.HasPrefix(s.Context, "pull-kubevirt-e2e") {
			hasE2E = true
		}
	}
	return laneStatus, hasE2E, nil
}

func checkRawStatuses(client ghClient, owner, repoName, sha, lane string) (hasURL bool, updatedAt string) {
	statuses, err := client.ListStatuses(owner, repoName, sha)
	if err != nil {
		logrus.WithError(err).Warn("failed to list raw statuses")
		return false, ""
	}
	for _, s := range statuses {
		if s.Context != lane {
			continue
		}
		// ListStatuses returns statuses in reverse-chronological order;
		// the first match for our lane is the latest.
		return s.TargetURL != "", ""
	}
	return false, ""
}

func buildStuckPR(pr *github.PullRequest, statusState, statusUpdatedAt string, hasURL bool) *StuckPR {
	var labels []string
	for _, l := range pr.Labels {
		labels = append(labels, l.Name)
	}
	return &StuckPR{
		Number:          pr.Number,
		Title:           pr.Title,
		Author:          pr.User.Login,
		HeadSHA:         pr.Head.SHA,
		UpdatedAt:       pr.UpdatedAt.Format(time.RFC3339),
		Labels:          labels,
		IsDraft:         pr.Draft,
		StatusState:     statusState,
		StatusUpdatedAt: statusUpdatedAt,
		HasTargetURL:    hasURL,
	}
}

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
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/go-github/v32/github"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
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
	scanCmd.Flags().StringVar(&scanLane, "lane", "", "required status check context name (required)")
	scanCmd.Flags().StringVarP(&scanOutput, "output", "o", "", "output YAML file path (default: stdout)")
	_ = scanCmd.MarkFlagRequired("lane")
}

func runScan(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()
	owner, repoName, err := parseRepo()
	if err != nil {
		return err
	}

	client, err := newGitHubClient(ctx)
	if err != nil {
		return err
	}

	log := logrus.WithFields(logrus.Fields{"lane": scanLane, "repo": repo})

	if err := verifyRequiredCheck(ctx, client, owner, repoName, scanLane); err != nil {
		return err
	}
	log.Info("confirmed lane is a required status check")

	prs, err := listOpenPRs(ctx, client, owner, repoName)
	if err != nil {
		return err
	}
	log.WithField("count", len(prs)).Info("fetched open PRs")

	summary, stuckPRs := classifyPRs(ctx, client, owner, repoName, scanLane, prs)

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

func verifyRequiredCheck(ctx context.Context, client *github.Client, owner, repoName, lane string) error {
	protection, _, err := client.Repositories.GetBranchProtection(ctx, owner, repoName, "main")
	if err != nil {
		return fmt.Errorf("fetching branch protection: %w", err)
	}
	if protection.RequiredStatusChecks == nil {
		return fmt.Errorf("no required status checks configured on main")
	}
	for _, check := range protection.RequiredStatusChecks.Contexts {
		if check == lane {
			return nil
		}
	}
	return fmt.Errorf("lane %q is not a required status check on main", lane)
}

func listOpenPRs(ctx context.Context, client *github.Client, owner, repoName string) ([]*github.PullRequest, error) {
	var allPRs []*github.PullRequest
	opts := &github.PullRequestListOptions{
		State:       "open",
		Base:        "main",
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		prs, resp, err := client.PullRequests.List(ctx, owner, repoName, opts)
		if err != nil {
			return nil, fmt.Errorf("listing PRs: %w", err)
		}
		allPRs = append(allPRs, prs...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return allPRs, nil
}

type prResult struct {
	category string // "stuck", "missing", "running", "success", "failed"
	pr       *StuckPR
}

func classifyPRs(ctx context.Context, client *github.Client, owner, repoName, lane string, prs []*github.PullRequest) (ScanSummary, []StuckPR) {
	const workers = 10
	sem := make(chan struct{}, workers)
	results := make([]prResult, len(prs))
	var wg sync.WaitGroup

	for i, pr := range prs {
		wg.Add(1)
		go func(idx int, pr *github.PullRequest) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx] = classifyPR(ctx, client, owner, repoName, lane, pr)
		}(i, pr)
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

func classifyPR(ctx context.Context, client *github.Client, owner, repoName, lane string, pr *github.PullRequest) prResult {
	log := logrus.WithField("pr", pr.GetNumber())
	sha := pr.GetHead().GetSHA()

	combinedStatus, hasE2E, err := getCombinedStatusInfo(ctx, client, owner, repoName, sha, lane)
	if err != nil {
		log.WithError(err).Warn("failed to get combined status")
		return prResult{category: "failed"}
	}

	if combinedStatus == nil {
		if !hasE2E {
			return prResult{category: "success"}
		}
		return prResult{category: "missing", pr: buildStuckPR(pr, "missing", "", false)}
	}

	switch combinedStatus.GetState() {
	case "success":
		return prResult{category: "success"}
	case "failure", "error":
		return prResult{category: "failed"}
	case "pending":
		hasURL, statusUpdatedAt := checkRawStatuses(ctx, client, owner, repoName, sha, lane)
		if hasURL {
			return prResult{category: "running"}
		}
		return prResult{category: "stuck", pr: buildStuckPR(pr, "pending", statusUpdatedAt, false)}
	default:
		return prResult{category: "failed"}
	}
}

func getCombinedStatusInfo(ctx context.Context, client *github.Client, owner, repoName, sha, lane string) (laneStatus *github.RepoStatus, hasE2E bool, err error) {
	opts := &github.ListOptions{PerPage: 100}
	for {
		combined, resp, err := client.Repositories.GetCombinedStatus(ctx, owner, repoName, sha, opts)
		if err != nil {
			return nil, false, err
		}
		for _, s := range combined.Statuses {
			if s.GetContext() == lane {
				laneStatus = s
			}
			if strings.HasPrefix(s.GetContext(), "pull-kubevirt-e2e") {
				hasE2E = true
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return laneStatus, hasE2E, nil
}

func checkRawStatuses(ctx context.Context, client *github.Client, owner, repoName, sha, lane string) (hasURL bool, updatedAt string) {
	opts := &github.ListOptions{PerPage: 100}
	var latest *github.RepoStatus
	for {
		statuses, resp, err := client.Repositories.ListStatuses(ctx, owner, repoName, sha, opts)
		if err != nil {
			logrus.WithError(err).Warn("failed to list raw statuses")
			return false, ""
		}
		for i := range statuses {
			if statuses[i].GetContext() != lane {
				continue
			}
			if latest == nil || statuses[i].GetUpdatedAt().After(latest.GetUpdatedAt()) {
				latest = statuses[i]
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	if latest == nil {
		return false, ""
	}
	return latest.GetTargetURL() != "", latest.GetUpdatedAt().Format(time.RFC3339)
}

func buildStuckPR(pr *github.PullRequest, statusState, statusUpdatedAt string, hasURL bool) *StuckPR {
	var labels []string
	for _, l := range pr.Labels {
		labels = append(labels, l.GetName())
	}
	return &StuckPR{
		Number:          pr.GetNumber(),
		Title:           pr.GetTitle(),
		Author:          pr.GetUser().GetLogin(),
		HeadSHA:         pr.GetHead().GetSHA(),
		UpdatedAt:       pr.GetUpdatedAt().Format(time.RFC3339),
		Labels:          labels,
		IsDraft:         pr.GetDraft(),
		StatusState:     statusState,
		StatusUpdatedAt: statusUpdatedAt,
		HasTargetURL:    hasURL,
	}
}

package github

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kubevirt.io/project-infra/pkg/flakefinder/api"

	"github.com/google/go-github/v28/github"
	"github.com/sirupsen/logrus"
)

type PullRequest struct {
	Number int
	SHA    string
}

func (p *PullRequest) ID() int {
	return p.Number
}

func (p *PullRequest) Matches(status *api.StartedStatus) bool {
	for _, v := range status.Repos {
		if strings.Contains(v, fmt.Sprintf("%d:%s", p.Number, p.SHA)) {
			return true
		}
	}
	return false
}

type Query struct {
	c          *github.Client
	org        string
	repo       string
	baseBranch string
}

func NewQuery(c *github.Client, org string, repo string, baseBranch string) *Query {
	return &Query{c: c, org: org, repo: repo, baseBranch: baseBranch}
}

func (q *Query) Query(ctx context.Context, startOfReport time.Time, endOfReport time.Time) ([]api.Change, error) {
	logrus.Infof("Fetching Prs from %v till %v", startOfReport, endOfReport)

	logrus.Infof("Filtering PRs for base branch %s", q.baseBranch)
	var changes []api.Change
	for nextPage := 1; nextPage > 0; {
		pullRequests, response, err := q.c.PullRequests.List(ctx, q.org, q.repo, &github.PullRequestListOptions{
			Base:        q.baseBranch,
			State:       "closed",
			Sort:        "updated",
			Direction:   "desc",
			ListOptions: github.ListOptions{Page: nextPage},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to fetch PRs for page %d: %v.", nextPage, err)
		}
		nextPage = response.NextPage
		for _, pr := range pullRequests {
			if startOfReport.After(*pr.UpdatedAt) {
				nextPage = 0
				break
			}
			if pr.MergedAt == nil {
				continue
			}
			if startOfReport.After(*pr.MergedAt) {
				continue
			}
			if endOfReport.Before(*pr.MergedAt) {
				continue
			}
			logrus.Infof("Adding PR %v '%v' (updated at %s)", *pr.Number, *pr.Title, pr.UpdatedAt.Format(time.RFC3339))
			changes = append(changes, &PullRequest{
				Number: *pr.Number,
				SHA:    *pr.Head.SHA,
			})
		}
	}
	logrus.Infof("%d pull requests found.", len(changes))
	return changes, nil
}

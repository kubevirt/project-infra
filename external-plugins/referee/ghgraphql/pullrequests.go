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

package ghgraphql

import (
	"context"
	"fmt"

	"github.com/shurcooL/githubv4"
)

func (g gitHubGraphQLClient) FetchOpenApprovedAndLGTMedPRs(org string, repo string) (PullRequests, error) {
	prs, err := g.fetchPRs(org, repo)
	if err != nil {
		return PullRequests{}, err
	}
	pullRequests := PullRequests{
		PRs: prs,
	}
	return pullRequests, nil
}

func (g gitHubGraphQLClient) fetchPRs(org string, repo string) ([]PullRequest, error) {
	var query struct {
		Repository struct {
			PullRequests struct {
				Nodes []PullRequest
			} `graphql:"pullRequests(first: 100, baseRefName: \"main\", labels: [\"lgtm\", \"approved\"], states: [OPEN])"`
		} `graphql:"repository(owner: $org, name: $repo)"`
	}
	variables := map[string]interface{}{
		"org":  githubv4.String(org),
		"repo": githubv4.String(repo),
	}

	err := g.gitHubClient.Query(context.Background(), &query, variables)
	if err != nil {
		return []PullRequest{}, fmt.Errorf("failed to use github query %+v with variables %v: %w", query, variables, err)
	}
	return query.Repository.PullRequests.Nodes, nil
}

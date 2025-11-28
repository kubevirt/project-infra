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

func (g gitHubGraphQLClient) FetchPRLabels(org string, repo string, prNumber int) (PRLabels, error) {
	labels, err := g.fetchLabelsForPR(org, repo, prNumber)
	if err != nil {
		return PRLabels{}, err
	}
	return NewPRLabels(labels), nil
}

func (g gitHubGraphQLClient) fetchLabelsForPR(org string, repo string, prNumber int) ([]Label, error) {
	var query struct {
		Repository struct {
			PullRequest struct {
				Labels Labels `graphql:"labels (first: 100)"`
			} `graphql:"pullRequest(number: $prNumber)"`
		} `graphql:"repository(owner: $org, name: $repo)"`
	}
	variables := map[string]interface{}{
		"prNumber": githubv4.Int(prNumber),
		"org":      githubv4.String(org),
		"repo":     githubv4.String(repo),
	}

	err := g.gitHubClient.Query(context.Background(), &query, variables)
	if err != nil {
		return []Label{}, fmt.Errorf("failed to use github query %+v with variables %v: %w", query, variables, err)
	}
	return query.Repository.PullRequest.Labels.Nodes, nil
}

func NewPRLabels(labels []Label) PRLabels {
	prLabels := PRLabels{
		Labels: labels,
	}
	for _, label := range labels {
		if label.Name == "do-not-merge/hold" {
			prLabels.IsHoldPresent = true
		}
	}
	return prLabels
}

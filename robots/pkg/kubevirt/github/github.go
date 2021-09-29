/*
 * Copyright 2021 The KubeVirt Authors.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package github

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"

	"kubevirt.io/project-infra/robots/pkg/kubevirt/cmd/flags"
)

func NewGitHubClient(ctx context.Context) (*github.Client, error) {
	var client *github.Client
	if flags.Options.GitHubTokenPath == "" {
		var err error
		client, err = github.NewEnterpriseClient(flags.Options.GitHubEndPoint, flags.Options.GitHubEndPoint, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create github client: %v", err)
		}
	} else {
		token, err := ioutil.ReadFile(flags.Options.GitHubTokenPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read github token file %s: %v", flags.Options.GitHubTokenPath, err)
		}
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: string(token)},
		)
		client, err = github.NewEnterpriseClient(flags.Options.GitHubEndPoint, flags.Options.GitHubEndPoint, oauth2.NewClient(ctx, ts))
		if err != nil {
			return nil, fmt.Errorf("failed to create github client: %v", err)
		}
	}
	return client, nil
}

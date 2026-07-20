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
 */

package testutils

import (
	"os"
	"path/filepath"
	"regexp"

	"k8s.io/apimachinery/pkg/util/sets"
	prowfakegithub "sigs.k8s.io/prow/pkg/github/fakegithub"
	"sigs.k8s.io/prow/pkg/layeredsets"
	"sigs.k8s.io/prow/pkg/plugins/ownersconfig"
	"sigs.k8s.io/prow/pkg/repoowners"
	"sigs.k8s.io/yaml"
)

const botName = prowfakegithub.Bot

const (
	// Bot is the exported botName
	Bot = botName
	// TestRef is the ref returned when calling GetRef
	TestRef = "abcde"
)

// FakeClient is like client, but fake.
type FakeClient = prowfakegithub.FakeClient

type FakeOwnersClient struct {
	ExistingTopLevelApprovers sets.Set[string]
	CurrentLeafApprovers      map[string]sets.Set[string]
	owners                    map[string]string
	approvers                 map[string]layeredsets.String
	reviewers                 map[string]layeredsets.String
	requiredReviewers         map[string]sets.Set[string]
	leafReviewers             map[string]sets.Set[string]
	dirBlacklist              []*regexp.Regexp
}

// FakeOwnersClient allows us to exercise ownership logic.
type FakeRepoownersClient struct {
	Foc *FakeOwnersClient
}

func (froc FakeRepoownersClient) LoadRepoOwners(org, repo, base string) (repoowners.RepoOwner, error) {
	return froc.Foc, nil
}

func (foc *FakeOwnersClient) TopLevelApprovers() sets.Set[string] {
	return foc.ExistingTopLevelApprovers
}

func (foc *FakeOwnersClient) Approvers(path string) layeredsets.String {
	return foc.approvers[path]
}

func (foc *FakeOwnersClient) LeafApprovers(path string) sets.Set[string] {
	return foc.CurrentLeafApprovers[path]
}

func (foc *FakeOwnersClient) FindApproverOwnersForFile(path string) string {
	return foc.owners[path]
}

func (foc *FakeOwnersClient) Reviewers(path string) layeredsets.String {
	return foc.reviewers[path]
}

func (foc *FakeOwnersClient) RequiredReviewers(path string) sets.Set[string] {
	return foc.requiredReviewers[path]
}

func (foc *FakeOwnersClient) LeafReviewers(path string) sets.Set[string] {
	return foc.leafReviewers[path]
}

func (foc *FakeOwnersClient) FindReviewersOwnersForFile(path string) string {
	return foc.owners[path]
}

func (foc *FakeOwnersClient) FindLabelsForFile(path string) sets.Set[string] {
	return sets.Set[string]{}
}

func (foc *FakeOwnersClient) IsNoParentOwners(path string) bool {
	return false
}

func (foc *FakeOwnersClient) Filenames() ownersconfig.Filenames {
	return ownersconfig.FakeFilenames
}

func (foc *FakeOwnersClient) IsAutoApproveUnownedSubfolders(path string) bool {
	return false
}

func (foc *FakeOwnersClient) ParseSimpleConfig(path string) (repoowners.SimpleConfig, error) {
	dir := filepath.Dir(path)
	for _, re := range foc.dirBlacklist {
		if re.MatchString(dir) {
			return repoowners.SimpleConfig{}, filepath.SkipDir
		}
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return repoowners.SimpleConfig{}, err
	}
	full := new(repoowners.SimpleConfig)
	err = yaml.Unmarshal(b, full)
	return *full, err
}

func (foc *FakeOwnersClient) ParseFullConfig(path string) (repoowners.FullConfig, error) {
	dir := filepath.Dir(path)
	for _, re := range foc.dirBlacklist {
		if re.MatchString(dir) {
			return repoowners.FullConfig{}, filepath.SkipDir
		}
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return repoowners.FullConfig{}, err
	}
	full := new(repoowners.FullConfig)
	err = yaml.Unmarshal(b, full)
	return *full, err
}

func (foc *FakeOwnersClient) AllOwners() sets.Set[string] {
	//TODO implement me
	panic("implement me")
}

func (foc *FakeOwnersClient) AllApprovers() sets.Set[string] {
	//TODO implement me
	panic("implement me")
}

func (foc *FakeOwnersClient) AllReviewers() sets.Set[string] {
	//TODO implement me
	panic("implement me")
}

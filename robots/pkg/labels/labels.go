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
 * Copyright the Kubernetes Authors.
 * Copyright the KubeVirt Authors.
 *
 */

package labels

import "time"

// originally from k8s.io/test-infra/label_sync

// LabelTarget specifies the intent of the label (PR or issue)
type LabelTarget string

const (
	prTarget    LabelTarget = "prs"
	issueTarget LabelTarget = "issues"
	bothTarget  LabelTarget = "both"
)

// Label holds declarative data about the label.
type Label struct {
	// Name is the current name of the label
	Name string `yaml:"name"`
	// Color is rrggbb or color
	Color string `yaml:"color"`
	// Description is brief text explaining its meaning, who can apply it
	Description string `yaml:"description,omitempty"`
	// Target specifies whether it targets PRs, issues or both
	Target LabelTarget `yaml:"target,omitempty"`
	// ProwPlugin specifies which prow plugin add/removes this label
	ProwPlugin string `yaml:"prowPlugin,omitempty"`
	// IsExternalPlugin specifies if the prow plugin is external or not
	IsExternalPlugin bool `yaml:"isExternalPlugin,omitempty"`
	// AddedBy specifies whether human/munger/bot adds the label
	AddedBy string `yaml:"addedBy,omitempty"`
	// Previously lists deprecated names for this label
	Previously []Label `yaml:"previously,omitempty"`
	// DeleteAfter specifies the label is retired and a safe date for deletion
	DeleteAfter *time.Time `yaml:"deleteAfter,omitempty"`
	parent      *Label     // Current name for previous labels (used internally)
}

// Configuration is a list of Repos defining Required Labels to sync into them
// There is also a Default list of labels applied to every Repo
type Configuration struct {
	Default RepoConfig            `yaml:"default"`
	Repos   map[string]RepoConfig `yaml:"repos,omitempty"`
	Orgs    map[string]RepoConfig `yaml:"orgs,omitempty"`
}

// RepoConfig contains only labels for the moment
type RepoConfig struct {
	Labels []Label `yaml:"labels"`
}

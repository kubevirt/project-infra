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
 * Copyright 2023 Red Hat, Inc.
 *
 */

package test_label_analyzer

import (
	"strings"

	"kubevirt.io/project-infra/pkg/git"
)

type TestFilesStats struct {
	FilesStats []*FileStats `json:"files_stats"`

	*Config `json:"config"`
}

// FileStats contains the information of the file whose outline was traversed and the results of the
// traversal.
type FileStats struct {
	*TestStats `json:"test_stats"`

	// RemoteURL is the absolute path to the file, most certainly an absolute URL inside a version control repository
	// containing a commit ID in order to exactly define the state of the file that was traversed
	RemoteURL string `json:"path"`
}

// TestStats contains the results of traversing a set of Ginkgo outlines and collecting the pathes in the form of
// GinkgoNode slices matching a Config describing the criteria to match against.
type TestStats struct {

	// SpecsTotal is the total number of specs encountered during traversal of the outline
	SpecsTotal int `json:"specs_total"`

	// MatchingSpecPaths is the slice of PathStats to matching specs for the collection of outlines traversed.
	// Each PathStats inside this slice is the path to each of the matching specs defined by the Config
	// being used. Please note that the GinkgoNode.Nodes are being removed during traversal.
	MatchingSpecPaths []*PathStats `json:"matching_spec_paths"`
}

// PathStats contains all relevant data to a path matching a Config.
type PathStats struct {

	// Lines has the line numbers for the matching nodes
	Lines []int `json:"lines"`

	// GitBlameLines is the output of the blame command for each of the Lines
	GitBlameLines []*git.BlameLine `json:"git_blame_lines"`

	// Path denotes the path to the spec that has been found to match
	Path []*GinkgoNode `json:"path"`

	// MatchingCategory holds the category that matched the path
	MatchingCategory *LabelCategory `json:"matching_category"`
}

func GetStatsFromGinkgoOutline(config *Config, gingkoOutline []*GinkgoNode) *TestStats {
	stats := &TestStats{
		SpecsTotal: 0,
	}
	traverseNodesRecursively(stats, config, gingkoOutline, nil)
	return stats
}

func traverseNodesRecursively(stats *TestStats, config *Config, gingkoOutline []*GinkgoNode, parents []*GinkgoNode) {
	for _, node := range gingkoOutline {
		var parentsWithNode []*GinkgoNode
		parentsWithNode = append(parentsWithNode, parents...)
		parentsWithNode = append(parentsWithNode, node)
		if node.Spec {
			stats.SpecsTotal++
			for _, category := range config.Categories {
				var testName string
				for _, nodeFromPath := range parentsWithNode {
					testName = strings.Join([]string{testName, nodeFromPath.Text}, " ")
				}
				if category.TestNameLabelRE.MatchString(testName) {
					var path []*GinkgoNode
					for _, pathNode := range parentsWithNode {
						path = append(path, pathNode.CloneWithoutNodes())
					}
					stats.MatchingSpecPaths = append(stats.MatchingSpecPaths, &PathStats{
						Path:             path,
						MatchingCategory: category,
					})
					category.Hits++
				}
			}
		}
		traverseNodesRecursively(stats, config, node.Nodes, parentsWithNode)
	}
}

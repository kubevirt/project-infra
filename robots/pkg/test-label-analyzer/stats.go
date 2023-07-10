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
	"regexp"
	"strconv"
	"strings"
	"time"
)

// FileStats contains the information of the file whose outline was traversed and the results of the
// traversal.
type FileStats struct {
	*Config    `json:"config"`
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
	GitBlameLines []*GitBlameInfo `json:"git_blame_lines"`

	// Path denotes the path to the spec that has been found to match
	Path []*GinkgoNode `json:"path"`
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
						Path: path,
					})
				}
			}
		}
		traverseNodesRecursively(stats, config, node.Nodes, parentsWithNode)
	}
}

const gitDateLayout = "2006-01-02 15:04:05 -0700"

var gitBlameRegex = regexp.MustCompile(`^([0-9a-f]+)(\s+\S+)?\s+\(([\S ]+)\s([0-9]{4}-[0-9]{2}-[0-9]{2}\s[0-9]{2}:[0-9]{2}:[0-9]{2}\s[-+][0-9]{4})\s+([0-9]+)\)\s(.*)$`)

type GitBlameInfo struct {
	CommitID string    `json:"commit_id"`
	Author   string    `json:"author"`
	Date     time.Time `json:"date"`
	LineNo   int       `json:"line_no"`
	Line     string    `json:"line"`
}

func ExtractGitBlameInfo(lines []string) []*GitBlameInfo {
	var info []*GitBlameInfo
	for _, line := range lines {
		if !gitBlameRegex.MatchString(line) {
			continue
		}
		submatches := gitBlameRegex.FindAllStringSubmatch(line, -1)
		date, err := time.Parse(gitDateLayout, submatches[0][4])
		if err != nil {
			panic(err)
		}
		lineNo, err := strconv.Atoi(submatches[0][5])
		if err != nil {
			panic(err)
		}
		info = append(info, &GitBlameInfo{
			CommitID: submatches[0][1],
			Author:   strings.TrimSpace(submatches[0][3]),
			Date:     date,
			LineNo:   lineNo,
			Line:     submatches[0][6],
		})
	}
	return info
}

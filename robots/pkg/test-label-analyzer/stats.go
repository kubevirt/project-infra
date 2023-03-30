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

type TestStats struct {
	SpecsTotal         int
	SpecsMatching      int
	MatchingSpecPathes [][]*GinkgoNode
}

func GetStatsFromGinkgoOutline(config *Config, gingkoOutline []*GinkgoNode) *TestStats {
	stats := &TestStats{
		SpecsTotal:    0,
		SpecsMatching: 0,
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
				testNameMatchesLabelRE := false
				for _, nodeFromPath := range parentsWithNode {
					testNameMatchesLabelRE = category.TestNameLabelRE.MatchString(nodeFromPath.Text)
					if testNameMatchesLabelRE {
						break
					}
				}
				if testNameMatchesLabelRE {
					stats.SpecsMatching++
					var path []*GinkgoNode
					for _, pathNode := range parentsWithNode {
						path = append(path, pathNode.CloneWithoutNodes())
					}
					stats.MatchingSpecPathes = append(stats.MatchingSpecPathes, path)
				}
			}
		}
		traverseNodesRecursively(stats, config, node.Nodes, parentsWithNode)
	}
}

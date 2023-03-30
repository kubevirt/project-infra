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

type GinkgoNode struct {
	Name    string        `json:"name"`
	Text    string        `json:"text"`
	Start   int           `json:"start"`
	End     int           `json:"end"`
	Spec    bool          `json:"spec"`
	Focused bool          `json:"focused"`
	Pending bool          `json:"pending"`
	Labels  []string      `json:"labels"`
	Nodes   []*GinkgoNode `json:"nodes"`
}

func (n GinkgoNode) CloneWithoutNodes() *GinkgoNode {
	return &GinkgoNode{
		Name:    n.Name,
		Text:    n.Text,
		Start:   n.Start,
		End:     n.End,
		Spec:    n.Spec,
		Focused: n.Focused,
		Pending: n.Pending,
		Labels:  n.Labels,
	}
}

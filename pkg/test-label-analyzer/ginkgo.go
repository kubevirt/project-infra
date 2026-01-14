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

// GinkgoNode is the basic structure that is compatible with the output of the command `ginkgo outline --format json`
type GinkgoNode struct {

	// Name of the node, usually `Describe`, `Context` ....
	Name string `json:"name"`

	// Text holds the description of the node
	Text string `json:"text"`

	// Start is the beginning of the textual representation of this Ginkgo node, i.e. the byte offset in the file from
	// which the outline originated from
	Start int `json:"start"`

	// End is the end of the textual representation of this Ginkgo node, i.e. the byte offset in the file from
	// which the outline originated from
	End int `json:"end"`

	// Spec denotes whether this is an actual Spec aka test or not
	Spec bool `json:"spec"`

	// Focused states whether the spec is focused
	Focused bool `json:"focused"`

	// Pending states whether the spec is pending
	Pending bool `json:"pending"`

	// Labels gives an array of the labels attached to the node
	Labels []string `json:"labels"`

	// Nodes holds the child nodes of this node
	Nodes []*GinkgoNode `json:"nodes"`
}

// CloneWithoutNodes creates a copy of this node excluding its children
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

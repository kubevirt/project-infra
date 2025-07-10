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

package ginkgo

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("stats", func() {

	Context("OutlineFromFile", func() {

		It("generates outline from file", func() {
			outline, err := OutlineFromFile("testdata/simple_test.ginkgo")
			Expect(err).ToNot(HaveOccurred())
			Expect(outline).ToNot(BeNil())
		})
		It("does not panic but just returns nil on outline from file missing import", func() {
			outline, err := OutlineFromFile("testdata/simple-basic_test.go")
			Expect(err).ToNot(HaveOccurred())
			Expect(outline).To(BeNil())
		})

		It("does not panic on outline from non existing file", func() {
			_, err := OutlineFromFile("testdata/nonexistent_test.go")
			Expect(err).To(HaveOccurred())
		})

	})

	Context("Node", func() {
		When("clone", func() {
			var n *Node
			BeforeEach(func() {
				n = &Node{
					Nodes: []*Node{
						{
							Name:    "By",
							Text:    "the way",
							Start:   0,
							End:     0,
							Spec:    false,
							Focused: false,
							Pending: false,
							Labels:  nil,
							Nodes:   nil,
						},
					},
				}
			})
			It("filters nodes", func() {
				Expect(n.CloneWithNodes(func(n *Node) bool {
					return n.Name != "By"
				}).Nodes).To(HaveLen(0))
			})
			It("doesn't filter other nodes", func() {
				Expect(n.CloneWithNodes(func(n *Node) bool {
					return n.Name == "By"
				}).Nodes).To(HaveLen(1))
			})
		})
	})

})

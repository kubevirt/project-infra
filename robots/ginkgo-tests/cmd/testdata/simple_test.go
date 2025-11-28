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
 * Copyright The KubeVirt Authors.
 *
 */

package simple

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("simple", func() {

	Context("was previously part of the latter node", func() {

		It("[test_id:1742]is still found", func() {
			Expect(DoesSomething(true)).ToNot(BeNil())
		})

	})

	When("does something is executed", func() {

		It("does not return an error", func() {
			Expect(DoesSomething(false)).To(BeNil())
		})

		It("does return an error", func() {
			Expect(DoesSomething(true)).ToNot(BeNil())
		})

		It("[vendor:cnv-qe]does do something else", func() {
			Expect(DoesSomething(true)).ToNot(BeNil())
		})

	})

	DescribeTable("is a table",
		func(a, b string) {
			Expect(a).ToNot(BeEquivalentTo(""))
			Expect(b).ToNot(BeEquivalentTo(""))
		},
		Entry("first testcase", "17", "42"),
		Entry("[test_id:8976]2nd testcase", "17", "42"),
	)

})

var _ = Describe(ExtendArgs("description of describe", func() {

	It("description of it", func() {
		Expect(DoesSomething(false)).To(BeNil())
	})

}))

func ExtendArgs(text string, args ...interface{}) (extendedText string, newArgs []interface{}) {
	return "whatever " + text, args
}

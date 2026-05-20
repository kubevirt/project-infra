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
	"go/ast"
	"os"
	"strings"
)

var _ = Describe("new test descriptor", func() {

	When("test 'does not return an error'", func() {
		var test *SourceTestDescriptor
		var err error

		BeforeEach(func() {
			test, err = NewTestDescriptorForName("simple does something is executed does not return an error", "testdata/simple_test.go")
		})

		It("returns no error", func() {
			Expect(err).To(BeNil())
		})

		It("returns an instance", func() {
			Expect(test).ToNot(BeNil())
		})

		It("initializes code", func() {
			Expect(test.FileCode()).ToNot(BeEquivalentTo(""))
		})

		It("initializes file", func() {
			Expect(test.File()).ToNot(BeNil())
		})

		It("initializes test outline node", func() {
			Expect(test.OutlineNode()).ToNot(BeNil())
		})

		It("initializes test ast", func() {
			Expect(test.Test()).ToNot(BeNil())
		})

		It("hits right test ast", func() {
			Expect(strings.Trim(test.Test().Args[0].(*ast.BasicLit).Value, "\"")).To(BeEquivalentTo("does not return an error"))
		})

	})

	When("test 'simple does something is executed [test_id:1742]is still found'", func() {
		var test *SourceTestDescriptor
		var err error

		BeforeEach(func() {
			test, err = NewTestDescriptorForID("simple does something is executed [test_id:1742]is still found", "testdata/simple_test.go")
		})

		It("returns no error", func() {
			Expect(err).To(BeNil())
		})

		It("returns an instance", func() {
			Expect(test).ToNot(BeNil())
		})

		It("initializes code", func() {
			Expect(test.FileCode()).ToNot(BeEquivalentTo(""))
		})

		It("initializes file", func() {
			Expect(test.File()).ToNot(BeNil())
		})

		It("initializes test outline node", func() {
			Expect(test.OutlineNode()).ToNot(BeNil())
		})

		It("initializes test ast", func() {
			Expect(test.Test()).ToNot(BeNil())
		})

		It("hits right test ast", func() {
			Expect(strings.Trim(test.Test().Args[0].(*ast.BasicLit).Value, "\"")).To(BeEquivalentTo("[test_id:1742]is still found"))
		})

	})

	When("test 'simple is a table [test_id:8976]2nd testcase'", func() {
		var test *SourceTestDescriptor
		var err error

		BeforeEach(func() {
			test, err = NewTestDescriptorForID("simple is a table [test_id:8976]2nd testcase", "testdata/simple_test.go")
		})

		It("returns no error", func() {
			Expect(err).To(BeNil())
		})

		It("returns an instance", func() {
			Expect(test).ToNot(BeNil())
		})

		It("initializes code", func() {
			Expect(test.FileCode()).ToNot(BeEquivalentTo(""))
		})

		It("initializes file", func() {
			Expect(test.File()).ToNot(BeNil())
		})

		It("initializes test outline node", func() {
			Expect(test.OutlineNode()).ToNot(BeNil())
		})

		It("initializes test ast", func() {
			Expect(test.Test()).ToNot(BeNil())
		})

		It("hits right test ast", func() {
			Expect(strings.Trim(test.Test().Args[0].(*ast.BasicLit).Value, "\"")).To(BeEquivalentTo("[test_id:8976]2nd testcase"))
		})

	})

	When("test 'does return an error'", func() {
		var test *SourceTestDescriptor

		BeforeEach(func() {
			var err error
			test, err = NewTestDescriptorForName("simple does something is executed does return an error", "testdata/simple_test.go")
			Expect(err).ToNot(HaveOccurred())
		})

		It("initializes for other test", func() {
			Expect(strings.Trim(test.Test().Args[0].(*ast.BasicLit).Value, "\"")).To(BeEquivalentTo("does return an error"))
		})

	})

	When("something goes wrong", func() {

		It("returns an os error if test file is not set", func() {
			_, err := NewTestDescriptorForName("some test", "")
			Expect(err).To(HaveOccurred())
		})

		It("returns an os error if test name is not set", func() {
			_, err := NewTestDescriptorForName("", "testdata/simple_test.go")
			Expect(err).To(HaveOccurred())
		})

		It("returns an os error if test file doesn't exist", func() {
			_, err := NewTestDescriptorForName("some test", "some/non/existing/file_test.go")
			Expect(os.IsNotExist(err)).To(BeTrue())
		})

		It("returns an error if the test does not exist", func() {
			_, err := NewTestDescriptorForName("some test that doesn't exist", "testdata/simple_test.go")
			Expect(err).ToNot(BeNil())
		})

	})

})

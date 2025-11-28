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
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("test-file", func() {
	When("FindTestFileByName", func() {
		It("finds a file if exact test name is given", func() {
			// Note: this test is contained in a When node which has the speciality
			// adding the when to the node test, therefore we have to handle that internally by adjusting the normalization
			// accordingly
			actualFilepath, err := FindTestFileByName("simple does something is executed does not return an error", "testdata")
			Expect(err).ToNot(HaveOccurred())
			stat, err := os.Stat(actualFilepath)
			Expect(err).ToNot(HaveOccurred())
			Expect(filepath.Base(stat.Name())).To(BeEquivalentTo("simple_test.go"))
		})
		It("finds same file if another exact test name is given", func() {
			actualFilepath, err := FindTestFileByName("simple does something is executed does return an error", "testdata")
			Expect(err).ToNot(HaveOccurred())
			stat, err := os.Stat(actualFilepath)
			Expect(err).ToNot(HaveOccurred())
			Expect(filepath.Base(stat.Name())).To(BeEquivalentTo("simple_test.go"))
		})
		It("finds same file if another exact test name with square brackets is given", func() {
			actualFilepath, err := FindTestFileByName("simple does something is executed [vendor:cnv-qe]does do something else", "testdata")
			Expect(err).ToNot(HaveOccurred())
			stat, err := os.Stat(actualFilepath)
			Expect(err).ToNot(HaveOccurred())
			Expect(filepath.Base(stat.Name())).To(BeEquivalentTo("simple_test.go"))
		})
		It("doesn't find a file if a non existing match is given", func() {
			_, err := FindTestFileByName("simple does something is executed does check for something else", "testdata")
			Expect(err).To(HaveOccurred())
		})
		It("finds a file with complex naming scheme", func() {
			actualFilepath, err := FindTestFileByName("[sig-network]SRIOV externalized was previously part of the latter node is still found", "testdata")
			Expect(err).ToNot(HaveOccurred())
			stat, err := os.Stat(actualFilepath)
			Expect(err).ToNot(HaveOccurred())
			Expect(filepath.Base(stat.Name())).To(BeEquivalentTo("externalized_test.go"))
		})
	})
	When("FindTestFileById", func() {
		It("finds file if node was moved inside a file but test_id is present", func() {
			actualFilepath, err := FindTestFileById("simple does something is executed [test_id:1742]is still found", "testdata")
			Expect(err).ToNot(HaveOccurred())
			stat, err := os.Stat(actualFilepath)
			Expect(err).ToNot(HaveOccurred())
			Expect(filepath.Base(stat.Name())).To(BeEquivalentTo("simple_test.go"))
		})
		It("finds file if node was moved into another file but test_id is present", func() {
			actualFilepath, err := FindTestFileById("simple does something is executed [test_id:4217]is still found", "testdata")
			Expect(err).ToNot(HaveOccurred())
			stat, err := os.Stat(actualFilepath)
			Expect(err).ToNot(HaveOccurred())
			Expect(filepath.Base(stat.Name())).To(BeEquivalentTo("simple-other_test.go"))
		})
		It("finds file if node is in table entry", func() {
			actualFilepath, err := FindTestFileById("simple is a table [test_id:8976]2nd testcase", "testdata")
			Expect(err).ToNot(HaveOccurred())
			stat, err := os.Stat(actualFilepath)
			Expect(err).ToNot(HaveOccurred())
			Expect(filepath.Base(stat.Name())).To(BeEquivalentTo("simple_test.go"))
		})
		It("err on no id present", func() {
			_, err := FindTestFileById("simple does something is executed does check for something else", "testdata")
			Expect(err).To(HaveOccurred())
		})
		It("err on no file with id present", func() {
			_, err := FindTestFileById("simple does something is executed [test_id:1737]is still found", "testdata")
			Expect(err).To(HaveOccurred())
		})
	})
})

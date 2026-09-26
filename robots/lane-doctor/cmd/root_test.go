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

package cmd

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("parseRepo", func() {

	DescribeTable("parses owner/repo format",
		func(input string, expectedOwner, expectedRepo string, expectErr bool) {
			repo = input
			owner, repoName, err := parseRepo()
			if expectErr {
				Expect(err).To(HaveOccurred())
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(owner).To(Equal(expectedOwner))
				Expect(repoName).To(Equal(expectedRepo))
			}
		},
		Entry("valid repo", "kubevirt/kubevirt", "kubevirt", "kubevirt", false),
		Entry("valid repo with different owner", "openshift/origin", "openshift", "origin", false),
		Entry("missing repo name", "kubevirt/", "", "", true),
		Entry("missing owner", "/kubevirt", "", "", true),
		Entry("no slash", "kubevirt", "", "", true),
		Entry("empty string", "", "", "", true),
		Entry("only slash", "/", "", "", true),
		Entry("extra slashes preserved in repo name", "owner/repo/extra", "owner", "repo/extra", false),
	)
})

var _ = Describe("writeOutput", func() {

	It("writes to a file when path is given", func() {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, "out.yaml")

		err := writeOutput([]byte("test-data"), path)
		Expect(err).NotTo(HaveOccurred())

		data, err := os.ReadFile(path)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(Equal("test-data"))
	})

	It("writes to stdout when no path is given", func() {
		err := writeOutput([]byte("stdout-data"), "")
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns wrapped error for invalid file path", func() {
		err := writeOutput([]byte("data"), "/nonexistent-dir/foo/bar.yaml")
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("writing output file"))
	})
})

var _ = Describe("resolveTokenPath", func() {

	It("returns the token path directly when set", func() {
		dir := GinkgoT().TempDir()
		path := filepath.Join(dir, "token")
		Expect(os.WriteFile(path, []byte("my-token"), 0644)).To(Succeed())

		tokenPath = path
		defer func() { tokenPath = "" }()

		resolved, err := resolveTokenPath()
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved).To(Equal(path))
	})

	It("creates a temp file from GITHUB_TOKEN env var", func() {
		tokenPath = ""
		GinkgoT().Setenv("GITHUB_TOKEN", "env-token")

		resolved, err := resolveTokenPath()
		Expect(err).NotTo(HaveOccurred())
		Expect(resolved).NotTo(BeEmpty())

		data, err := os.ReadFile(resolved)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(data)).To(Equal("env-token"))
	})

	It("returns error when no token source is available", func() {
		tokenPath = ""
		GinkgoT().Setenv("GITHUB_TOKEN", "")

		_, err := resolveTokenPath()
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("no GitHub token"))
	})
})

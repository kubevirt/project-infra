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

package git

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("git log", func() {
	When("fetching log commits", Ordered, func() {

		var commits []*LogCommit

		BeforeAll(func() {
			var err error
			commits, err = LogCommits("main..HEAD", "testdata/repo", "tests/")
			Expect(err).ToNot(HaveOccurred())
		})

		It("has commits", func() {
			Expect(len(commits)).ToNot(BeEquivalentTo(0))
		})

		It("has commits with hashes", func() {
			Expect(commits[0].Hash).ToNot(BeEquivalentTo(""))
		})

		It("has commits with file changes", func() {
			Expect(len(commits[0].FileChanges)).ToNot(BeEquivalentTo(0))
		})

		It("has file changes with type", func() {
			Expect(commits[0].FileChanges[0].ChangeType).ToNot(BeEquivalentTo(""))
		})

		It("has file changes with filename", func() {
			Expect(commits[0].FileChanges[0].Filename).ToNot(BeEquivalentTo(""))
		})

	})

	When("parsing git output", func() {
		DescribeTable("fileChangeRegex matches",
			func(outputLineToMatch string) {
				Expect(fileChangeRegex.MatchString(outputLineToMatch)).To(BeTrue())
			},
			Entry("matches M", "M\ttests/virtctl/BUILD.bazel"),
			Entry("matches D", "D\ttests/virtctl/ssh.go"),
			Entry("matches R", "R066\ttests/virtctl/scp.go\ttests/virtctl/ssh_scp.go"),
		)
		DescribeTable("fileChangeRegex captures",
			func(outputLineToMatch, expectedModifier, expectedSimilarity, expectedFilename, expectedFilename2 string) {
				submatches := fileChangeRegex.FindStringSubmatch(outputLineToMatch)
				Expect(submatches[1]).To(BeEquivalentTo(expectedModifier))
				Expect(submatches[2]).To(BeEquivalentTo(expectedSimilarity))
				Expect(submatches[3]).To(BeEquivalentTo(expectedFilename))
				// submatches[4] is a match containing the whitespace also, so we don't use it
				Expect(submatches[5]).To(BeEquivalentTo(expectedFilename2))
			},
			Entry("captures M",
				"M\ttests/virtctl/BUILD.bazel",
				"M",
				"",
				"tests/virtctl/BUILD.bazel",
				"",
			),
			Entry("captures D",
				"D\ttests/virtctl/ssh.go",
				"D",
				"",
				"tests/virtctl/ssh.go",
				"",
			),
			Entry("captures R",
				"R066\ttests/virtctl/scp.go\ttests/virtctl/ssh_scp.go",
				"R",
				"066",
				"tests/virtctl/scp.go",
				"tests/virtctl/ssh_scp.go",
			),
		)
	})
})

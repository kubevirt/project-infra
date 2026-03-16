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
 */

package filter

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("expressions", func() {
	Context("buildExpressions", func() {
		It("returns empty expressions for empty input", func() {
			out, err := buildExpressions(nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(out.Filter).To(BeEmpty())
			Expect(out.Skip).To(BeEmpty())
			Expect(out.LabelFilter).To(BeEmpty())
		})

		It("deduplicates matches and escapes regex metacharacters", func() {
			matches := matchingTests{
				{
					Id:       "id-1",
					Reason:   "sig-network",
					Version:  "",
					TestName: "test [1]",
				},
				{
					Id:       "id-2",
					Reason:   "sig-storage",
					Version:  "",
					TestName: "test .2",
				},
				{
					Id:       "id-3",
					Reason:   "sig-network",
					Version:  "",
					TestName: "test [1]",
				},
			}

			out, err := buildExpressions(matches)
			Expect(err).ToNot(HaveOccurred())

			Expect(out.Skip).To(BeEmpty())
			Expect(out.Filter).To(Equal(`test \.2|test \[1\]`))
			Expect(out.LabelFilter).To(Equal("sig-network||sig-storage"))
		})

		It("returns an error when a reason is not a valid Ginkgo label", func() {
			_, err := buildExpressions(matchingTests{
				{
					Id:       "id-1",
					Reason:   "flaky test - Tracked in https://github.com/kubevirt/kubevirt/issues/37",
					Version:  "",
					TestName: "test A",
				},
			})

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot generate --label-filter"))
		})
	})

	Context("runExpressions", func() {
		It("writes text output to the provided writer", func() {
			tempDir, err := os.MkdirTemp("", "expressions")
			Expect(err).ToNot(HaveOccurred())
			defer os.RemoveAll(tempDir)

			inputPath := filepath.Join(tempDir, "matching-tests.json")
			matches := matchingTests{
				{
					Id:       "id-1",
					Reason:   "sig-network",
					Version:  "",
					TestName: "test [1]",
				},
				{
					Id:       "id-2",
					Reason:   "sig-storage",
					Version:  "",
					TestName: "test .2",
				},
			}

			payload, err := json.Marshal(matches)
			Expect(err).ToNot(HaveOccurred())
			Expect(os.WriteFile(inputPath, payload, 0600)).To(Succeed())

			var out bytes.Buffer
			err = runExpressions(&filterExpressionsOptions{
				inputFilePath: inputPath,
				mode:          "text",
			}, &out)
			Expect(err).ToNot(HaveOccurred())
			Expect(out.String()).To(Equal("# Ginkgo v1\n--filter=test \\.2|test \\[1\\]\n\n# Ginkgo v2\n--label-filter=sig-network||sig-storage\n"))
		})
	})
})

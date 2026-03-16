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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("expressions", func() {
	Context("buildExpressions", func() {
		It("returns empty expressions for empty input", func() {
			out := buildExpressions(nil)
			Expect(out.Filter).To(BeEmpty())
			Expect(out.Skip).To(BeEmpty())
			Expect(out.LabelFilter).To(BeEmpty())
		})

		It("builds expressions from matching tests", func() {
			matches := matchingTests{
				{
					Id:       "id-1",
					Reason:   "Quarantine",
					Version:  "",
					TestName: "test A",
				},
				{
					Id:       "id-2",
					Reason:   "Quarantine",
					Version:  "",
					TestName: "test B",
				},
			}

			out := buildExpressions(matches)

			Expect(out.Skip).To(BeEmpty())

			// Filter should mention both test names
			Expect(out.Filter).To(ContainSubstring("test A"))
			Expect(out.Filter).To(ContainSubstring("test B"))

			// LabelFilter should mention the reason once
			Expect(strings.Count(out.LabelFilter, "Quarantine")).To(Equal(1))
		})
	})
})


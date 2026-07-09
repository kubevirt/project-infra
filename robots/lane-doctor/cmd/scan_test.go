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
	"time"

	"github.com/google/go-github/v32/github"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func strPtr(s string) *string   { return &s }
func intPtr(i int) *int         { return &i }
func boolPtr(b bool) *bool      { return &b }

var _ = Describe("buildStuckPR", func() {

	It("extracts fields from a GitHub PullRequest", func() {
		updated := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
		pr := &github.PullRequest{
			Number: intPtr(42),
			Title:  strPtr("Fix flaky test"),
			User:   &github.User{Login: strPtr("dhiller")},
			Head:   &github.PullRequestBranch{SHA: strPtr("abc123")},
			Labels: []*github.Label{
				{Name: strPtr("lgtm")},
				{Name: strPtr("approved")},
			},
			Draft:     boolPtr(false),
			UpdatedAt: &updated,
		}

		result := buildStuckPR(pr, "pending", "2026-06-10T11:00:00Z", false)

		Expect(result.Number).To(Equal(42))
		Expect(result.Title).To(Equal("Fix flaky test"))
		Expect(result.Author).To(Equal("dhiller"))
		Expect(result.HeadSHA).To(Equal("abc123"))
		Expect(result.Labels).To(ConsistOf("lgtm", "approved"))
		Expect(result.IsDraft).To(BeFalse())
		Expect(result.StatusState).To(Equal("pending"))
		Expect(result.StatusUpdatedAt).To(Equal("2026-06-10T11:00:00Z"))
		Expect(result.HasTargetURL).To(BeFalse())
	})

	It("handles a draft PR with no labels", func() {
		updated := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
		pr := &github.PullRequest{
			Number:    intPtr(99),
			Title:     strPtr("WIP: something"),
			User:      &github.User{Login: strPtr("user")},
			Head:      &github.PullRequestBranch{SHA: strPtr("def456")},
			Labels:    nil,
			Draft:     boolPtr(true),
			UpdatedAt: &updated,
		}

		result := buildStuckPR(pr, "missing", "", true)

		Expect(result.Number).To(Equal(99))
		Expect(result.IsDraft).To(BeTrue())
		Expect(result.Labels).To(BeNil())
		Expect(result.StatusState).To(Equal("missing"))
		Expect(result.StatusUpdatedAt).To(BeEmpty())
		Expect(result.HasTargetURL).To(BeTrue())
	})
})

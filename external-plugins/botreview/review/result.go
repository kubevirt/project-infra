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
 */

package review

import (
	"fmt"

	"github.com/sourcegraph/go-diff/diff"
)

// BotReviewResult describes the interface of a review result for a pull request.
type BotReviewResult interface {
	String() string

	// IsApproved states if the review has only expected changes
	IsApproved() bool

	// ShouldNotMergeReason returns the reason why the pull request should not get merged without a human review, if any
	ShouldNotMergeReason() string

	// AddReviewFailure stores the data of a hunk of code that failed review
	AddReviewFailure(fileName string, hunks ...*diff.Hunk)

	// ShortString provides a short description of the review result
	ShortString() string
}

func NewCanMergeReviewResult(approveComment string, disapproveComment string) BotReviewResult {
	return &BasicReviewResult{
		approveComment:    approveComment,
		disapproveComment: disapproveComment,
	}
}

func NewShouldNotMergeReviewResult(approveComment string, disapproveComment string, reason string) BotReviewResult {
	return &BasicReviewResult{
		approveComment:       approveComment,
		disapproveComment:    disapproveComment,
		shouldNotMergeReason: reason,
	}
}

func newReviewResultWithData(approveComment string, disapproveComment string, notMatchingHunks map[string][]*diff.Hunk, shouldNotMergeReason string) BotReviewResult {
	return &BasicReviewResult{
		approveComment:       approveComment,
		disapproveComment:    disapproveComment,
		notMatchingHunks:     notMatchingHunks,
		shouldNotMergeReason: shouldNotMergeReason,
	}
}

// BasicReviewResult is the default implementation to store the relevant data for creation of a PR comment.
type BasicReviewResult struct {
	approveComment       string
	disapproveComment    string
	notMatchingHunks     map[string][]*diff.Hunk
	shouldNotMergeReason string
}

func (r *BasicReviewResult) String() string {
	if r.IsApproved() {
		return r.approveComment
	} else {
		comment := r.disapproveComment
		comment += "\n\n<details>\n"
		for fileName, hunks := range r.notMatchingHunks {
			comment += fmt.Sprintf("\n_%s_", fileName)
			for _, hunk := range hunks {
				comment += fmt.Sprintf("\n\n~~~diff\n%s\n~~~", string(hunk.Body))
			}
		}
		comment += "\n\n</details>\n"
		return comment
	}
}

func (r *BasicReviewResult) IsApproved() bool {
	return len(r.notMatchingHunks) == 0
}

func (r *BasicReviewResult) ShouldNotMergeReason() string {
	return r.shouldNotMergeReason
}

func (r *BasicReviewResult) AddReviewFailure(fileName string, hunks ...*diff.Hunk) {
	if r.notMatchingHunks == nil {
		r.notMatchingHunks = make(map[string][]*diff.Hunk)
	}
	if _, exists := r.notMatchingHunks[fileName]; !exists {
		r.notMatchingHunks[fileName] = hunks
	} else {
		r.notMatchingHunks[fileName] = append(r.notMatchingHunks[fileName], hunks...)
	}
}

func (r *BasicReviewResult) ShortString() string {
	if r.IsApproved() {
		return r.approveComment
	} else {
		comment := r.disapproveComment
		comment += "\n  <details>"
		for fileName := range r.notMatchingHunks {
			comment += fmt.Sprintf("\n  * `%s`", fileName)
		}
		comment += "\n  </details>"
		return comment
	}
}

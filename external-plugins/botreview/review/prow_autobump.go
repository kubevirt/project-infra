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
 * Copyright the KubeVirt authors.
 *
 */

package review

import (
	"fmt"
	"github.com/sourcegraph/go-diff/diff"
	"regexp"
	"strings"
)

const (
	prowAutobumpApproveComment    = `:thumbsup: This looks like a simple prow autobump.`
	prowAutobumpDisapproveComment = `:thumbsdown: This doesn't look like a simple prow autobump.

I found suspicious hunks:
`
)

var prowAutobumpHunkBodyMatcher *regexp.Regexp

func init() {
	prowAutobumpHunkBodyMatcher = regexp.MustCompile(`(?m)^(-[\s]+- image: [^\s]+$[\n]^\+[\s]+- image: [^\s]+|-[\s]+(image|clonerefs|initupload|entrypoint|sidecar): [^\s]+$[\n]^\+[\s]+(image|clonerefs|initupload|entrypoint|sidecar): [^\s]+)$`)
}

type ProwAutobumpResult struct {
	notMatchingHunks map[string][]*diff.Hunk
}

func (r ProwAutobumpResult) String() string {
	if len(r.notMatchingHunks) == 0 {
		return prowAutobumpApproveComment
	} else {
		comment := prowAutobumpDisapproveComment
		for fileName, hunks := range r.notMatchingHunks {
			comment += fmt.Sprintf("\n%s", fileName)
			for _, hunk := range hunks {
				comment += fmt.Sprintf("\n```\n%s\n```", string(hunk.Body))
			}
		}
		return comment
	}
}

func (r ProwAutobumpResult) IsApproved() bool {
	return len(r.notMatchingHunks) == 0
}

func (r ProwAutobumpResult) CanMerge() bool {
	return false
}

func (r *ProwAutobumpResult) AddReviewFailure(fileName string, hunks ...*diff.Hunk) {
	if r.notMatchingHunks == nil {
		r.notMatchingHunks = make(map[string][]*diff.Hunk)
	}
	if _, exists := r.notMatchingHunks[fileName]; !exists {
		r.notMatchingHunks[fileName] = hunks
	} else {
		r.notMatchingHunks[fileName] = append(r.notMatchingHunks[fileName], hunks...)
	}
}

func (r ProwAutobumpResult) ShortString() string {
	if r.IsApproved() {
		return prowAutobumpApproveComment
	} else {
		comment := prowAutobumpDisapproveComment
		comment += fmt.Sprintf("\nFiles:")
		for fileName := range r.notMatchingHunks {
			comment += fmt.Sprintf("\n* `%s`", fileName)
		}
		return comment
	}
}

type ProwAutobump struct {
	relevantFileDiffs []*diff.FileDiff
	notMatchingHunks  []*diff.Hunk
}

func (t *ProwAutobump) IsRelevant() bool {
	return len(t.relevantFileDiffs) > 0
}

func (t *ProwAutobump) AddIfRelevant(fileDiff *diff.FileDiff) {
	fileName := strings.TrimPrefix(fileDiff.NewName, "b/")

	if !strings.HasPrefix(fileName, "github/ci/prow-deploy/kustom") {
		return
	}

	t.relevantFileDiffs = append(t.relevantFileDiffs, fileDiff)
}

func (t *ProwAutobump) Review() BotReviewResult {
	result := &ProwAutobumpResult{}

	for _, fileDiff := range t.relevantFileDiffs {
		fileName := strings.TrimPrefix(fileDiff.NewName, "b/")
		for _, hunk := range fileDiff.Hunks {
			match := prowAutobumpHunkBodyMatcher.Match(hunk.Body)
			if !match {
				result.AddReviewFailure(fileName, hunk)
			}
		}
	}

	return result
}

func (t *ProwAutobump) String() string {
	return fmt.Sprintf("relevantFileDiffs: %v", t.relevantFileDiffs)
}

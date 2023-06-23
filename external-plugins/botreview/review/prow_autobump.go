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
 * Copyright 2022 Red Hat, Inc.
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

These are the suspicious hunks I found:
`
)

var prowAutobumpHunkBodyMatcher *regexp.Regexp

func init() {
	prowAutobumpHunkBodyMatcher = regexp.MustCompile(`(?m)^(-[\s]+- image: [^\s]+$[\n]^\+[\s]+- image: [^\s]+|-[\s]+(image|clonerefs|initupload|entrypoint|sidecar): [^\s]+$[\n]^\+[\s]+(image|clonerefs|initupload|entrypoint|sidecar): [^\s]+)$`)
}

type ProwAutobumpResult struct {
	notMatchingHunks []*diff.Hunk
}

func (r ProwAutobumpResult) String() string {
	if len(r.notMatchingHunks) == 0 {
		return prowAutobumpApproveComment
	} else {
		comment := prowAutobumpDisapproveComment
		for _, hunk := range r.notMatchingHunks {
			comment += fmt.Sprintf("\n```\n%s\n```", string(hunk.Body))
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
		for _, hunk := range fileDiff.Hunks {
			if !prowAutobumpHunkBodyMatcher.Match(hunk.Body) {
				result.notMatchingHunks = append(result.notMatchingHunks, hunk)
			}
		}
	}

	return result
}

func (t *ProwAutobump) String() string {
	return fmt.Sprintf("relevantFileDiffs: %v", t.relevantFileDiffs)
}

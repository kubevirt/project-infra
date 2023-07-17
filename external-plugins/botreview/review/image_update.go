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
	prowJobImageUpdateApproveComment    = `:thumbsup: This looks like a simple prow job image bump.`
	prowJobImageUpdateDisapproveComment = `:thumbsdown: This doesn't look like a simple prow job image bump.

I found suspicious hunks:
`
)

var (
	prowJobImageUpdateHunkBodyMatcher   = regexp.MustCompile(`(?m)^(-[\s]+- image: [^\s]+$[\n]^\+[\s]+- image: [^\s]+|-[\s]+image: [^\s]+$[\n]^\+[\s]+image: [^\s]+)$`)
	prowJobReleaseBranchFileNameMatcher = regexp.MustCompile(`.*\/[\w-]+-[0-9-\.]+\.yaml`)
)

type ProwJobImageUpdateResult struct {
	notMatchingHunks map[string][]*diff.Hunk
}

func (r ProwJobImageUpdateResult) String() string {
	if r.IsApproved() {
		return prowJobImageUpdateApproveComment
	} else {
		comment := prowJobImageUpdateDisapproveComment
		for fileName, hunks := range r.notMatchingHunks {
			comment += fmt.Sprintf("\nFile: `%s`", fileName)
			for _, hunk := range hunks {
				comment += fmt.Sprintf("\n```\n%s\n```", string(hunk.Body))
			}
		}
		return comment
	}
}

func (r ProwJobImageUpdateResult) IsApproved() bool {
	return len(r.notMatchingHunks) == 0
}

func (r ProwJobImageUpdateResult) CanMerge() bool {
	return true
}

func (r *ProwJobImageUpdateResult) AddReviewFailure(fileName string, hunks ...*diff.Hunk) {

}

func (r ProwJobImageUpdateResult) ShortString() string {
	if r.IsApproved() {
		return prowJobImageUpdateApproveComment
	} else {
		comment := prowJobImageUpdateDisapproveComment
		comment += fmt.Sprintf("\nFiles:")
		for fileName := range r.notMatchingHunks {
			comment += fmt.Sprintf("\n* `%s`", fileName)
		}
		return comment
	}
}

type ProwJobImageUpdate struct {
	relevantFileDiffs []*diff.FileDiff
	notMatchingHunks  []*diff.Hunk
}

func (t *ProwJobImageUpdate) IsRelevant() bool {
	return len(t.relevantFileDiffs) > 0
}

func (t *ProwJobImageUpdate) AddIfRelevant(fileDiff *diff.FileDiff) {
	fileName := strings.TrimPrefix(fileDiff.NewName, "b/")

	// disregard all files
	//	* where the path is not beyond the jobconfig path
	//	* where the name changed and
	//  * who are not yaml
	if strings.TrimPrefix(fileDiff.OrigName, "a/") != fileName ||
		!strings.HasSuffix(fileName, ".yaml") ||
		!strings.HasPrefix(fileName, "github/ci/prow-deploy/files/jobs") ||
		prowJobReleaseBranchFileNameMatcher.MatchString(fileName) {
		return
	}

	t.relevantFileDiffs = append(t.relevantFileDiffs, fileDiff)
}

func (t *ProwJobImageUpdate) Review() BotReviewResult {
	result := &ProwJobImageUpdateResult{}

	for _, fileDiff := range t.relevantFileDiffs {
		fileName := strings.TrimPrefix(fileDiff.NewName, "b/")
		for _, hunk := range fileDiff.Hunks {
			if !prowJobImageUpdateHunkBodyMatcher.Match(hunk.Body) {
				result.AddReviewFailure(fileName, hunk)
			}
		}
	}

	return result
}

func (t *ProwJobImageUpdate) String() string {
	return fmt.Sprintf("relevantFileDiffs: %v", t.relevantFileDiffs)
}

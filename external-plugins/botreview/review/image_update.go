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
	"regexp"
	"strings"

	"github.com/sourcegraph/go-diff/diff"
)

const (
	prowJobImageUpdateApproveComment    = `:thumbsup: This looks like a simple prow job image bump.`
	prowJobImageUpdateDisapproveComment = `:thumbsdown: This doesn't look like a simple prow job image bump.`
)

var (
	prowJobImageUpdateLineMatcher       = regexp.MustCompile(`(?m)^\+\s+- image: \S+$`)
	prowJobImageUpdateHunkBodyMatcher   = regexp.MustCompile(`(?m)^(-\s+- image:\s+\S+$\n^\+\s+- image:\s+\S+|-\s+image:\s+\S+$\n^\+\s+image:\s+\S+)$`)
	prowJobReleaseBranchFileNameMatcher = regexp.MustCompile(`.*/[\w-]+-[0-9-.]+\.yaml`)
)

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
	//  * who match the release branch file name pattern
	if strings.TrimPrefix(fileDiff.OrigName, "a/") != fileName ||
		!strings.HasSuffix(fileName, ".yaml") ||
		!strings.HasPrefix(fileName, "github/ci/prow-deploy/files/jobs") ||
		prowJobReleaseBranchFileNameMatcher.MatchString(fileName) {
		return
	}

	// do a quick scan for image changes in hunks
	foundImageUpdate := false
	for _, hunk := range fileDiff.Hunks {
		if prowJobImageUpdateLineMatcher.Match(hunk.Body) {
			foundImageUpdate = true
			break
		}
	}
	// skip diff if none found
	if !foundImageUpdate {
		return
	}

	t.relevantFileDiffs = append(t.relevantFileDiffs, fileDiff)
}

func (t *ProwJobImageUpdate) Review() BotReviewResult {
	result := NewCanMergeReviewResult(prowJobImageUpdateApproveComment, prowJobImageUpdateDisapproveComment)

	for _, fileDiff := range t.relevantFileDiffs {
		fileName := strings.TrimPrefix(fileDiff.NewName, "b/")
		for _, hunk := range fileDiff.Hunks {
			if !t.hunkMatches(hunk) {
				result.AddReviewFailure(fileName, hunk)
			}
		}
	}

	return result
}

func (t *ProwJobImageUpdate) hunkMatches(hunk *diff.Hunk) bool {
	return prowJobImageUpdateHunkBodyMatcher.Match(hunk.Body)
}

func (t *ProwJobImageUpdate) String() string {
	return fmt.Sprintf("relevantFileDiffs: %v", t.relevantFileDiffs)
}

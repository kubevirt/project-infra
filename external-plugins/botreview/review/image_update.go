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
	prowJobImageUpdateApproveComment = `This looks like a simple prow job image bump. The bot approves.

/lgtm
/approve
`
	prowJobImageUpdateDisapproveComment = `This doesn't look like a simple prow job image bump.

These are the suspicious hunks I found:
`
)

var prowJobImageUpdateHunkBodyMatcher *regexp.Regexp

func init() {
	prowJobImageUpdateHunkBodyMatcher = regexp.MustCompile(`(?m)^(-[\s]+- image: [^\s]+$[\n]^\+[\s]+- image: [^\s]+|-[\s]+image: [^\s]+$[\n]^\+[\s]+image: [^\s]+)$`)
}

type Result struct {
	notMatchingHunks []*diff.Hunk
}

func (r Result) String() string {
	if len(r.notMatchingHunks) == 0 {
		return prowJobImageUpdateApproveComment
	} else {
		comment := prowJobImageUpdateDisapproveComment
		for _, hunk := range r.notMatchingHunks {
			comment += fmt.Sprintf("\n```\n%s\n```", string(hunk.Body))
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
		!strings.HasPrefix(fileName, "github/ci/prow-deploy/files/jobs") {
		return
	}

	t.relevantFileDiffs = append(t.relevantFileDiffs, fileDiff)
}

func (t *ProwJobImageUpdate) Review() BotReviewResult {
	result := &Result{}

	for _, fileDiff := range t.relevantFileDiffs {
		for _, hunk := range fileDiff.Hunks {
			if !prowJobImageUpdateHunkBodyMatcher.Match(hunk.Body) {
				result.notMatchingHunks = append(result.notMatchingHunks, hunk)
			}
		}
	}

	return result
}

func (t *ProwJobImageUpdate) String() string {
	return fmt.Sprintf("relevantFileDiffs: %v", t.relevantFileDiffs)
}

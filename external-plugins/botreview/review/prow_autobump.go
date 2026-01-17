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
	prowAutobumpApproveComment       = `:thumbsup: This looks like a simple prow autobump.`
	prowAutobumpDisapproveComment    = `:thumbsdown: This doesn't look like a simple prow autobump.`
	prowAutoBumpShouldNotMergeReason = "prow update should be merged at a point in time where it doesn't interfere with normal CI usage"
)

var prowAutobumpHunkBodyMatcher *regexp.Regexp
var prowAutobumpTitleMatcher *regexp.Regexp

func init() {
    prowAutobumpHunkBodyMatcher = regexp.MustCompile(`(?m)^(-[\s]+- image: [^\s]+$[\n]^\+[\s]+- image: [^\s]+|-[\s]+(image|clonerefs|initupload|entrypoint|sidecar): [^\s]+$[\n]^\+[\s]+(image|clonerefs|initupload|entrypoint|sidecar): [^\s]+)$`)
    prowAutobumpTitleMatcher = regexp.MustCompile(`(?i)\b(run\s+hack/(bump-prow\.sh|bump-prow-deployment-images\.sh)|bump\s+prow\s+deployment\s+images|bump\s+prow-deploy\s+images)\b`)
}

type ProwAutobump struct {
	relevantFileDiffs []*diff.FileDiff
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
	result := NewShouldNotMergeReviewResult(prowAutobumpApproveComment, prowAutobumpDisapproveComment, prowAutoBumpShouldNotMergeReason)

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

func (t *ProwAutobump) MatchSubject(subject string) bool {
	return prowAutobumpTitleMatcher.MatchString(subject)
}

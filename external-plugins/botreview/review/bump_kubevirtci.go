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
	BumpKubevirtCIApproveComment = `This looks like a simple prow job image bump. The bot approves.

/lgtm
/approve
`
	BumpKubevirtCIDisapproveComment = `This doesn't look like a simple prow job image bump.

These are the suspicious hunks I found:
`
)

var bumpKubevirtCIHunkBodyMatcher *regexp.Regexp

func init() {
	bumpKubevirtCIHunkBodyMatcher = regexp.MustCompile(`(?m)^(-[\s]+- image: [^\s]+$[\n]^\+[\s]+- image: [^\s]+|-[\s]+image: [^\s]+$[\n]^\+[\s]+image: [^\s]+)$`)
}

type BumpKubevirtCI struct {
	relevantFileDiffs []*diff.FileDiff
	notMatchingHunks  []*diff.Hunk
}

func (t *BumpKubevirtCI) IsRelevant() bool {
	return len(t.relevantFileDiffs) > 0
}

func (t *BumpKubevirtCI) AddIfRelevant(fileDiff *diff.FileDiff) {
	fileName := strings.TrimPrefix(fileDiff.NewName, "b/")

	// disregard all files
	//	* where the full path is not cluster-up-sha.txt and
	//	* where the path is not below cluster-up/
	if fileName != "cluster-up-sha.txt" || !strings.HasPrefix(fileName, "cluster-up/") {
		return
	}

	t.relevantFileDiffs = append(t.relevantFileDiffs, fileDiff)
}

func (t *BumpKubevirtCI) Review() BotReviewResult {
	result := &Result{}

	for _, fileDiff := range t.relevantFileDiffs {
		for _, hunk := range fileDiff.Hunks {
			if !bumpKubevirtCIHunkBodyMatcher.Match(hunk.Body) {
				result.notMatchingHunks = append(result.notMatchingHunks, hunk)
			}
		}
	}

	return result
}

func (t *BumpKubevirtCI) String() string {
	return fmt.Sprintf("relevantFileDiffs: %v", t.relevantFileDiffs)
}

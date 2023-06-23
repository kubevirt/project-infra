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
	bumpKubevirtCIApproveComment    = `:thumbsup: This looks like a simple kubevirtci bump.`
	bumpKubevirtCIDisapproveComment = `:thumbsdown: This doesn't look like a simple kubevirtci bump.

These are the suspicious hunks I found:
`
)

var bumpKubevirtCIHackConfigDefaultMatcher *regexp.Regexp
var bumpKubevirtCIClusterUpShaMatcher *regexp.Regexp
var bumpKubevirtCIClusterUpVersionMatcher *regexp.Regexp

func init() {
	bumpKubevirtCIHackConfigDefaultMatcher = regexp.MustCompile(`(?m)^-[\s]*kubevirtci_git_hash=\"[^\s]+\"$[\n]^\+[\s]*kubevirtci_git_hash=\"[^\s]+\"$`)
	bumpKubevirtCIClusterUpShaMatcher = regexp.MustCompile(`(?m)^-[\s]*[^\s]+$[\n]^\+[^\s]+$`)
	bumpKubevirtCIClusterUpVersionMatcher = regexp.MustCompile(`(?m)^-[0-9]+-[a-z0-9]+$[\n]^\+[0-9]+-[a-z0-9]+$`)
}

type BumpKubevirtCIResult struct {
	notMatchingHunks []*diff.Hunk
}

func (r BumpKubevirtCIResult) IsApproved() bool {
	return len(r.notMatchingHunks) == 0
}

func (r BumpKubevirtCIResult) CanMerge() bool {
	return true
}

func (r BumpKubevirtCIResult) String() string {
	if len(r.notMatchingHunks) == 0 {
		return bumpKubevirtCIApproveComment
	} else {
		comment := bumpKubevirtCIDisapproveComment
		for _, hunk := range r.notMatchingHunks {
			comment += fmt.Sprintf("\n```\n%s\n```", string(hunk.Body))
		}
		return comment
	}
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

	// store all hunks for unwanted files
	if fileName != "cluster-up-sha.txt" &&
		fileName != "hack/config-default.sh" &&
		!strings.HasPrefix(fileName, "cluster-up/") {
		for _, hunk := range fileDiff.Hunks {
			t.notMatchingHunks = append(t.notMatchingHunks, hunk)
		}
		return
	}

	t.relevantFileDiffs = append(t.relevantFileDiffs, fileDiff)
}

func (t *BumpKubevirtCI) Review() BotReviewResult {
	result := &BumpKubevirtCIResult{}

	for _, fileDiff := range t.relevantFileDiffs {
		fileName := strings.TrimPrefix(fileDiff.NewName, "b/")
		switch fileName {
		case "cluster-up-sha.txt":
			for _, hunk := range fileDiff.Hunks {
				if !bumpKubevirtCIClusterUpShaMatcher.Match(hunk.Body) {
					result.notMatchingHunks = append(result.notMatchingHunks, hunk)
				}
			}
		case "hack/config-default.sh":
			for _, hunk := range fileDiff.Hunks {
				if !bumpKubevirtCIHackConfigDefaultMatcher.Match(hunk.Body) {
					result.notMatchingHunks = append(result.notMatchingHunks, hunk)
				}
			}
		case "cluster-up/version.txt":
			for _, hunk := range fileDiff.Hunks {
				if !bumpKubevirtCIClusterUpVersionMatcher.Match(hunk.Body) {
					result.notMatchingHunks = append(result.notMatchingHunks, hunk)
				}
			}
		default:
			// no checks since we can't do anything reasonable here
		}
	}

	result.notMatchingHunks = append(result.notMatchingHunks, t.notMatchingHunks...)

	return result
}

func (t *BumpKubevirtCI) String() string {
	return fmt.Sprintf("relevantFileDiffs: %v", t.relevantFileDiffs)
}

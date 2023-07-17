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

I found suspicious hunks:
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
	notMatchingHunks map[string][]*diff.Hunk
}

func (r BumpKubevirtCIResult) IsApproved() bool {
	return len(r.notMatchingHunks) == 0
}

func (r BumpKubevirtCIResult) CanMerge() bool {
	return true
}

func (r BumpKubevirtCIResult) String() string {
	if r.IsApproved() {
		return bumpKubevirtCIApproveComment
	} else {
		comment := bumpKubevirtCIDisapproveComment
		for fileName, hunks := range r.notMatchingHunks {
			comment += fmt.Sprintf("\nFile: `%s`", fileName)
			for _, hunk := range hunks {
				comment += fmt.Sprintf("\n```\n%s\n```", string(hunk.Body))
			}
		}
		return comment
	}
}

func (r *BumpKubevirtCIResult) AddReviewFailure(fileName string, hunks ...*diff.Hunk) {
	if r.notMatchingHunks == nil {
		r.notMatchingHunks = make(map[string][]*diff.Hunk)
	}
	if _, exists := r.notMatchingHunks[fileName]; !exists {
		r.notMatchingHunks[fileName] = hunks
	} else {
		r.notMatchingHunks[fileName] = append(r.notMatchingHunks[fileName], hunks...)
	}
}

func (r BumpKubevirtCIResult) ShortString() string {
	if r.IsApproved() {
		return bumpKubevirtCIApproveComment
	} else {
		comment := bumpKubevirtCIDisapproveComment
		comment += fmt.Sprintf("\nFiles:")
		for fileName := range r.notMatchingHunks {
			comment += fmt.Sprintf("\n* `%s`", fileName)
		}
		return comment
	}
}

type BumpKubevirtCI struct {
	relevantFileDiffs []*diff.FileDiff
	unwantedFiles     map[string][]*diff.Hunk
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
			if t.unwantedFiles == nil {
				t.unwantedFiles = make(map[string][]*diff.Hunk, 0)
			}
			_, exists := t.unwantedFiles[fileName]
			if !exists {
				t.unwantedFiles[fileName] = []*diff.Hunk{hunk}
			} else {
				t.unwantedFiles[fileName] = append(t.unwantedFiles[fileName], hunk)
			}
		}
		return
	}

	t.relevantFileDiffs = append(t.relevantFileDiffs, fileDiff)
}

func (t *BumpKubevirtCI) Review() BotReviewResult {
	result := &BumpKubevirtCIResult{}

	for _, fileDiff := range t.relevantFileDiffs {
		fileName := strings.TrimPrefix(fileDiff.NewName, "b/")
		var matcher *regexp.Regexp
		switch fileName {
		case "cluster-up-sha.txt":
			matcher = bumpKubevirtCIClusterUpShaMatcher
		case "hack/config-default.sh":
			matcher = bumpKubevirtCIHackConfigDefaultMatcher
		case "cluster-up/version.txt":
			matcher = bumpKubevirtCIClusterUpVersionMatcher
		default:
			// no checks since we can't do anything reasonable here
			continue
		}
		if matcher != nil {
			for _, hunk := range fileDiff.Hunks {
				if !matcher.Match(hunk.Body) {
					result.AddReviewFailure(fileDiff.NewName, hunk)
				}
			}
		}
	}

	for fileName, unwantedFiles := range t.unwantedFiles {
		result.AddReviewFailure(fileName, unwantedFiles...)
	}

	return result
}

func (t *BumpKubevirtCI) String() string {
	return fmt.Sprintf("relevantFileDiffs: %v", t.relevantFileDiffs)
}

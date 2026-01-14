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
	bumpKubevirtCIApproveComment    = `:thumbsup: This looks like a simple kubevirtci bump.`
	bumpKubevirtCIDisapproveComment = `:thumbsdown: This doesn't look like a simple kubevirtci bump.`
)

var bumpKubevirtCIHackConfigDefaultMatcher *regexp.Regexp
var bumpKubevirtCIClusterUpShaMatcher *regexp.Regexp
var bumpKubevirtCIClusterUpVersionMatcher *regexp.Regexp

func init() {
	bumpKubevirtCIHackConfigDefaultMatcher = regexp.MustCompile(`(?m)^-[\s]*kubevirtci_git_hash=\"[^\s]+\"$[\n]^\+[\s]*kubevirtci_git_hash=\"[^\s]+\"$`)
	bumpKubevirtCIClusterUpShaMatcher = regexp.MustCompile(`(?m)^-[\s]*[^\s]+$[\n]^\+[^\s]+$`)
	bumpKubevirtCIClusterUpVersionMatcher = regexp.MustCompile(`(?m)^-[0-9]+-[a-z0-9]+$[\n]^\+[0-9]+-[a-z0-9]+$`)
}

type BumpKubevirtCI struct {
	relevantFileDiffs []*diff.FileDiff
	unwantedFiles     map[string]struct{}
}

func (t *BumpKubevirtCI) IsRelevant() bool {
	return len(t.relevantFileDiffs) > 0
}

func (t *BumpKubevirtCI) AddIfRelevant(fileDiff *diff.FileDiff) {
	fileName := strings.TrimPrefix(fileDiff.NewName, "b/")

	// handle deleted files
	wasDeleted := fileName == "/dev/null"
	if wasDeleted {
		fileName = strings.TrimPrefix(fileDiff.OrigName, "a/")
	}

	if fileName == "cluster-up-sha.txt" ||
		fileName == "hack/config-default.sh" ||
		strings.HasPrefix(fileName, "cluster-up/") {
		t.relevantFileDiffs = append(t.relevantFileDiffs, fileDiff)
		return
	}

	if t.unwantedFiles == nil {
		t.unwantedFiles = make(map[string]struct{})
	}
	t.unwantedFiles[fileName] = struct{}{}
}

func (t *BumpKubevirtCI) Review() BotReviewResult {
	result := NewCanMergeReviewResult(bumpKubevirtCIApproveComment, bumpKubevirtCIDisapproveComment)

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

	for fileName := range t.unwantedFiles {
		result.AddReviewFailure(fileName)
	}

	return result
}

func (t *BumpKubevirtCI) String() string {
	return fmt.Sprintf("relevantFileDiffs: %v", t.relevantFileDiffs)
}

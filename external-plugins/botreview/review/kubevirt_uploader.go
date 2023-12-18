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
	kubevirtUploaderApproveComment    = `:thumbsup: This looks like a simple kubevirt uploader bump.`
	kubevirtUploaderDisapproveComment = `:thumbsdown: This doesn't look like a kubevirt uploader bump.`
)

var kubevirtUploaderMatcher *regexp.Regexp

func init() {
	kubevirtUploaderMatcher = regexp.MustCompile(`(?m)^\s+"http:\S+$\n^\+\s+"https://storage.googleapis.com/builddeps\S+$`)
}

type KubeVirtUploader struct {
	relevantFileDiffs []*diff.FileDiff
	unwantedFiles     map[string]struct{}
}

func (t *KubeVirtUploader) IsRelevant() bool {
	return len(t.relevantFileDiffs) > 0
}

func (t *KubeVirtUploader) AddIfRelevant(fileDiff *diff.FileDiff) {
	fileName := strings.TrimPrefix(fileDiff.NewName, "b/")

	if fileName == "WORKSPACE" {
		t.relevantFileDiffs = append(t.relevantFileDiffs, fileDiff)
		return
	}

	if t.unwantedFiles == nil {
		t.unwantedFiles = make(map[string]struct{})
	}
	t.unwantedFiles[fileName] = struct{}{}
}

func (t *KubeVirtUploader) Review() BotReviewResult {
	result := NewCanMergeReviewResult(kubevirtUploaderApproveComment, kubevirtUploaderDisapproveComment)

	for _, fileDiff := range t.relevantFileDiffs {
		fileName := strings.TrimPrefix(fileDiff.NewName, "b/")
		var matcher *regexp.Regexp
		switch fileName {
		case "WORKSPACE":
			matcher = kubevirtUploaderMatcher
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

func (t *KubeVirtUploader) String() string {
	return fmt.Sprintf("relevantFileDiffs: %v", t.relevantFileDiffs)
}

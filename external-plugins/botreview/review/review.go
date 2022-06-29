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
)

type KindOfChange interface {
	AddIfRelevant(fileDiff *diff.FileDiff)
	Review() BotReviewResult
	IsRelevant() bool
}

type BotReviewResult interface {
	String() string
}

func newPossibleReviewTypes() []KindOfChange {
	return []KindOfChange{
		&ProwJobImageUpdate{},
		&BumpKubevirtCI{},
	}
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

func GuessReviewTypes(fileDiffs []*diff.FileDiff) []KindOfChange {
	possibleReviewTypes := newPossibleReviewTypes()
	for _, fileDiff := range fileDiffs {
		for _, kindOfChange := range possibleReviewTypes {
			kindOfChange.AddIfRelevant(fileDiff)
		}
	}
	result := []KindOfChange{}
	for _, t := range possibleReviewTypes {
		if t.IsRelevant() {
			result = append(result, t)
		}
	}
	return result
}

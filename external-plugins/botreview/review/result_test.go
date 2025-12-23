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
 * Copyright the KubeVirt Authors.
 */

package review

import (
	"testing"

	"github.com/sourcegraph/go-diff/diff"
)

func TestBasicReviewResult_ShortString(t *testing.T) {
	type fields struct {
		approveComment       string
		disapproveComment    string
		notMatchingHunks     map[string][]*diff.Hunk
		shouldNotMergeReason string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "empty not matching hunks should only return the approve comment",
			fields: fields{
				approveComment:       "approved",
				disapproveComment:    "disapproved",
				notMatchingHunks:     nil,
				shouldNotMergeReason: "",
			},
			want: "approved",
		},
		{
			name: "short string should be a bullet list of files",
			fields: fields{
				approveComment:    "approved",
				disapproveComment: "disapproved",
				notMatchingHunks: map[string][]*diff.Hunk{
					"meh": {{Body: []byte("blah")}},
				},
				shouldNotMergeReason: "",
			},
			want: `disapproved
  <details>
  * ` + "`meh`" + `
  </details>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &BasicReviewResult{
				approveComment:       tt.fields.approveComment,
				disapproveComment:    tt.fields.disapproveComment,
				notMatchingHunks:     tt.fields.notMatchingHunks,
				shouldNotMergeReason: tt.fields.shouldNotMergeReason,
			}
			if got := r.ShortString(); got != tt.want {
				t.Errorf("ShortString() = %v, want %v", got, tt.want)
			}
		})
	}
}

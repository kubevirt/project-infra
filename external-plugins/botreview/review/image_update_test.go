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
	"github.com/sourcegraph/go-diff/diff"
	"os"
	"reflect"
	"testing"
)

func TestProwJobImageUpdate_Review(t1 *testing.T) {
	diffFilePathes := []string{
		"testdata/simple_bump-prow-job-images_sh.patch0",
		"testdata/simple_bump-prow-job-images_sh.patch1",
		"testdata/mixed_bump_prow_job.patch0",
	}
	diffFilePathesToDiffs := map[string]*diff.FileDiff{}
	for _, diffFile := range diffFilePathes {
		bump_images_diff_file, err := os.ReadFile(diffFile)
		if err != nil {
			t1.Errorf("failed to read diff: %v", err)
		}
		bump_file_diffs, err := diff.ParseFileDiff(bump_images_diff_file)
		if err != nil {
			t1.Errorf("failed to read diff: %v", err)
		}
		diffFilePathesToDiffs[diffFile] = bump_file_diffs
	}
	type fields struct {
		relevantFileDiffs []*diff.FileDiff
	}
	tests := []struct {
		name   string
		fields fields
		want   *Result
	}{
		{
			name: "simple image bump",
			fields: fields{
				relevantFileDiffs: []*diff.FileDiff{
					diffFilePathesToDiffs["testdata/simple_bump-prow-job-images_sh.patch0"],
					diffFilePathesToDiffs["testdata/simple_bump-prow-job-images_sh.patch1"],
				},
			},
			want: &Result{},
		},
		{
			name: "mixed image bump",
			fields: fields{
				relevantFileDiffs: []*diff.FileDiff{
					diffFilePathesToDiffs["testdata/mixed_bump_prow_job.patch0"],
				},
			},
			want: &Result{
				notMatchingHunks: []*diff.Hunk{
					diffFilePathesToDiffs["testdata/mixed_bump_prow_job.patch0"].Hunks[0],
				},
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &ProwJobImageUpdate{
				relevantFileDiffs: tt.fields.relevantFileDiffs,
			}
			if got := t.Review(); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("Review() = %v, want %v", got, tt.want)
			}
		})
	}
}

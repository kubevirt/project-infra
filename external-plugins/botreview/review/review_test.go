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
	"github.com/sourcegraph/go-diff/diff"
	"os"
	"reflect"
	"testing"
)

func TestGuessReviewTypes(t *testing.T) {
	diffFilePaths := []string{
		"testdata/simple_bump-prow-job-images_sh.patch0",
		"testdata/simple_bump-prow-job-images_sh.patch1",
		"testdata/move_prometheus_stack.patch0",
		"testdata/move_prometheus_stack.patch1",
		"testdata/cdi_arm_release.patch0",
	}
	diffFilePathsToDiffs := map[string]*diff.FileDiff{}
	for _, diffFile := range diffFilePaths {
		bumpImagesDiffFile, err := os.ReadFile(diffFile)
		if err != nil {
			t.Errorf("failed to read diff: %v", err)
		}
		bumpFileDiffs, err := diff.ParseFileDiff(bumpImagesDiffFile)
		if err != nil {
			t.Errorf("failed to read diff: %v", err)
		}
		diffFilePathsToDiffs[diffFile] = bumpFileDiffs
	}
	type args struct {
		fileDiffs []*diff.FileDiff
	}
	tests := []struct {
		name string
		args args
		want []KindOfChange
	}{
		{
			name: "simple image bump should yield a change",
			args: args{
				fileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/simple_bump-prow-job-images_sh.patch0"],
					diffFilePathsToDiffs["testdata/simple_bump-prow-job-images_sh.patch1"],
				},
			},
			want: []KindOfChange{
				&ProwJobImageUpdate{
					relevantFileDiffs: []*diff.FileDiff{
						diffFilePathsToDiffs["testdata/simple_bump-prow-job-images_sh.patch0"],
						diffFilePathsToDiffs["testdata/simple_bump-prow-job-images_sh.patch1"],
					},
				},
			},
		},
		{
			name: "mixed with image bump should yield a partial change",
			args: args{
				fileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/simple_bump-prow-job-images_sh.patch0"],
					diffFilePathsToDiffs["testdata/move_prometheus_stack.patch0"],
				},
			},
			want: []KindOfChange{
				&ProwJobImageUpdate{
					relevantFileDiffs: []*diff.FileDiff{
						diffFilePathsToDiffs["testdata/simple_bump-prow-job-images_sh.patch0"],
					},
				},
			},
		},
		{
			name: "non image bump should not yield a change",
			args: args{
				fileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/move_prometheus_stack.patch0"],
					diffFilePathsToDiffs["testdata/move_prometheus_stack.patch1"],
				},
			},
			want: []KindOfChange{},
		},
		{
			name: "non image bump should not yield a change 2",
			args: args{
				fileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/cdi_arm_release.patch0"],
				},
			},
			want: []KindOfChange{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GuessReviewTypes(tt.args.fileDiffs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GuessReviewTypes() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
		"testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch0",
		"testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch1",
		"testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch2",
		"testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch3",
		"testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch4",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch00",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch01",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch02",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch03",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch04",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch05",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch06",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch07",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch08",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch09",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch10",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch11",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch12",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch13",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch14",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch15",
		"testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch16",
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
			name: "non image bump (move_prometheus_stack) should not yield a change",
			args: args{
				fileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/move_prometheus_stack.patch0"],
					diffFilePathsToDiffs["testdata/move_prometheus_stack.patch1"],
				},
			},
			want: nil,
		},
		{
			name: "non image bump (cdi_arm_release) should not yield a change 2",
			args: args{
				fileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/cdi_arm_release.patch0"],
				},
			},
			want: nil,
		},
		{
			name: "non kubevirtci bump (eviction-strategy-doc) should not yield a change",
			args: args{
				fileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch0"],
					diffFilePathsToDiffs["testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch1"],
					diffFilePathsToDiffs["testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch2"],
					diffFilePathsToDiffs["testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch3"],
					diffFilePathsToDiffs["testdata/kubevirt/eviction-strategy-doc/eviction-strategy-doc.patch4"],
				},
			},
			want: nil,
		},
		{
			name: "non kubevirtci bump (fix-containerdisks-migration) should not yield a change",
			args: args{
				fileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch00"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch01"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch02"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch03"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch04"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch05"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch06"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch07"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch08"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch09"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch10"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch11"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch12"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch13"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch14"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch15"],
					diffFilePathsToDiffs["testdata/kubevirt/fix-containerdisks-migration/fix-containerdisks-migrations.patch16"],
				},
			},
			want: nil,
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

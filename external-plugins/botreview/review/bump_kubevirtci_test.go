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
	"github.com/sourcegraph/go-diff/diff"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestBumpKubevirtCI_Review(t1 *testing.T) {
	diffFilePaths := []string{}
	entries, err := os.ReadDir("testdata/kubevirtci-bump")
	if err != nil {
		t1.Errorf("failed to read files: %v", err)
	}
	for _, entry := range entries {
		diffFilePaths = append(diffFilePaths, filepath.Join("testdata/kubevirtci-bump", entry.Name()))
	}
	diffFilePaths = append(diffFilePaths, "testdata/mixed_bump_prow_job.patch0")
	diffFilePathsToDiffs := map[string]*diff.FileDiff{}
	for _, diffFile := range diffFilePaths {
		bumpImagesDiffFile, err := os.ReadFile(diffFile)
		if err != nil {
			t1.Errorf("failed to read diff: %v", err)
		}
		bumpFileDiffs, err := diff.ParseFileDiff(bumpImagesDiffFile)
		if err != nil {
			t1.Errorf("failed to read diff: %v", err)
		}
		diffFilePathsToDiffs[diffFile] = bumpFileDiffs
	}
	type fields struct {
		relevantFileDiffs []*diff.FileDiff
	}
	tests := []struct {
		name   string
		fields fields
		want   BotReviewResult
	}{
		{
			name: "simple prow autobump",
			fields: fields{
				relevantFileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/kubevirtci-bump/cluster-up-sha.txt"],
					diffFilePathsToDiffs["testdata/kubevirtci-bump/cluster-up_cluster_kind-1.22-sriov_provider.sh"],
					diffFilePathsToDiffs["testdata/kubevirtci-bump/cluster-up_cluster_kind-1.22-sriov_sriov-node_node.sh"],
					diffFilePathsToDiffs["testdata/kubevirtci-bump/cluster-up_cluster_kind_common.sh"],
					diffFilePathsToDiffs["testdata/kubevirtci-bump/cluster-up_cluster_kind_configure-registry-proxy.sh"],
					diffFilePathsToDiffs["testdata/kubevirtci-bump/cluster-up_hack_common.sh"],
					diffFilePathsToDiffs["testdata/kubevirtci-bump/cluster-up_version.txt"],
					diffFilePathsToDiffs["testdata/kubevirtci-bump/hack_config-default.sh"],
				},
			},
			want: newReviewResultWithData(bumpKubevirtCIApproveComment, bumpKubevirtCIDisapproveComment, nil, true, ""),
		},
		{
			name: "mixed image bump",
			fields: fields{
				relevantFileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/kubevirtci-bump/cluster-up-sha.txt"],
					diffFilePathsToDiffs["testdata/kubevirtci-bump/cluster-up_cluster_kind-1.22-sriov_provider.sh"],
					diffFilePathsToDiffs["testdata/mixed_bump_prow_job.patch0"],
				},
			},
			want: newReviewResultWithData(bumpKubevirtCIApproveComment, bumpKubevirtCIDisapproveComment, map[string][]*diff.Hunk{"github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-presubmits.yaml": diffFilePathsToDiffs["testdata/mixed_bump_prow_job.patch0"].Hunks}, true, ""),
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &BumpKubevirtCI{}
			for _, fileDiff := range tt.fields.relevantFileDiffs {
				t.AddIfRelevant(fileDiff)
			}
			if got := t.Review(); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("Review() = %v, want %v", got, tt.want)
			}
		})
	}
}

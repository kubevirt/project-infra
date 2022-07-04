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
	"path/filepath"
	"reflect"
	"testing"
)

func TestBumpKubevirtCI_Review(t1 *testing.T) {
	diffFilePathes := []string{}
	entries, err := os.ReadDir("testdata/kubevirtci-bump")
	if err != nil {
		t1.Errorf("failed to read files: %v", err)
	}
	for _, entry := range entries {
		diffFilePathes = append(diffFilePathes, filepath.Join("testdata/kubevirtci-bump", entry.Name()))
	}
	diffFilePathes = append(diffFilePathes, "testdata/mixed_bump_prow_job.patch0")
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
		want   *BumpKubevirtCIResult
	}{
		{
			name: "simple prow autobump",
			fields: fields{
				relevantFileDiffs: []*diff.FileDiff{
					diffFilePathesToDiffs["testdata/kubevirtci-bump/cluster-up-sha.txt"],
					diffFilePathesToDiffs["testdata/kubevirtci-bump/cluster-up_cluster_kind-1.22-sriov_provider.sh"],
					diffFilePathesToDiffs["testdata/kubevirtci-bump/cluster-up_cluster_kind-1.22-sriov_sriov-node_node.sh"],
					diffFilePathesToDiffs["testdata/kubevirtci-bump/cluster-up_cluster_kind_common.sh"],
					diffFilePathesToDiffs["testdata/kubevirtci-bump/cluster-up_cluster_kind_configure-registry-proxy.sh"],
					diffFilePathesToDiffs["testdata/kubevirtci-bump/cluster-up_hack_common.sh"],
					diffFilePathesToDiffs["testdata/kubevirtci-bump/cluster-up_version.txt"],
					diffFilePathesToDiffs["testdata/kubevirtci-bump/hack_config-default.sh"],
				},
			},
			want: &BumpKubevirtCIResult{},
		},
		{
			name: "mixed image bump",
			fields: fields{
				relevantFileDiffs: []*diff.FileDiff{
					diffFilePathesToDiffs["testdata/kubevirtci-bump/cluster-up-sha.txt"],
					diffFilePathesToDiffs["testdata/kubevirtci-bump/cluster-up_cluster_kind-1.22-sriov_provider.sh"],
					diffFilePathesToDiffs["testdata/mixed_bump_prow_job.patch0"],
				},
			},
			want: &BumpKubevirtCIResult{
				notMatchingHunks: diffFilePathesToDiffs["testdata/mixed_bump_prow_job.patch0"].Hunks,
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &BumpKubevirtCI{}
			for _, diff := range tt.fields.relevantFileDiffs {
				t.AddIfRelevant(diff)
			}
			if got := t.Review(); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("Review() = %v, want %v", got, tt.want)
			}
		})
	}
}

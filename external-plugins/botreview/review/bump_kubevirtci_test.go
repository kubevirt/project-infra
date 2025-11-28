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
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/sourcegraph/go-diff/diff"
)

func TestBumpKubevirtCI_Review(t1 *testing.T) {
	diffFilePaths := []string{
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
		"testdata/kubevirt/ci-bump-remove-provider-local/bump-kci.patch00",
		"testdata/kubevirt/ci-bump-remove-provider-local/bump-kci.patch01",
		"testdata/kubevirt/ci-bump-remove-provider-local/bump-kci.patch02",
		"testdata/kubevirt/ci-bump-remove-provider-local/bump-kci.patch03",
		"testdata/kubevirt/ci-bump-remove-provider-local/bump-kci.patch04",
		"testdata/kubevirt/ci-bump-remove-provider-local/bump-kci.patch05",
	}
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
		if bumpFileDiffs == nil {
			panic(fmt.Sprintf("file diff %q empty", diffFile))
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
			name: "simple kubevirtci-bump",
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
			want: newReviewResultWithData(bumpKubevirtCIApproveComment, bumpKubevirtCIDisapproveComment, nil, ""),
		},
		{
			name: "mixed kubevirtci-bump",
			fields: fields{
				relevantFileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/kubevirtci-bump/cluster-up-sha.txt"],
					diffFilePathsToDiffs["testdata/kubevirtci-bump/cluster-up_cluster_kind-1.22-sriov_provider.sh"],
					diffFilePathsToDiffs["testdata/mixed_bump_prow_job.patch0"],
				},
			},
			want: newReviewResultWithData(bumpKubevirtCIApproveComment, bumpKubevirtCIDisapproveComment, map[string][]*diff.Hunk{"github/ci/prow-deploy/files/jobs/kubevirt/kubevirt/kubevirt-presubmits.yaml": nil}, ""),
		},
		{
			name: "non kubevirtci bump",
			fields: fields{
				relevantFileDiffs: []*diff.FileDiff{
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
			want: newReviewResultWithData(bumpKubevirtCIApproveComment, bumpKubevirtCIDisapproveComment, map[string][]*diff.Hunk{
				"pkg/virt-operator/resource/generate/components/daemonsets.go":            nil,
				"pkg/container-disk/container-disk_test.go":                               nil,
				"pkg/virt-handler/container-disk/BUILD.bazel":                             nil,
				"staging/src/kubevirt.io/api/core/v1/types_swagger_generated.go":          nil,
				"staging/src/kubevirt.io/client-go/api/openapi_generated.go":              nil,
				"staging/src/kubevirt.io/api/core/v1/types.go":                            nil,
				"api/openapi-spec/swagger.json":                                           nil,
				"pkg/virt-handler/container-disk/mount.go":                                nil,
				"pkg/virt-controller/services/template.go":                                nil,
				"staging/src/kubevirt.io/api/core/v1/deepcopy_generated.go":               nil,
				"pkg/virt-operator/resource/generate/components/validations_generated.go": nil,
				"pkg/virt-handler/vm.go":                                                  nil,
				"pkg/virt-handler/container-disk/generated_mock_mount.go":                 nil,
				"pkg/virt-handler/isolation/isolation.go":                                 nil,
				"pkg/virt-handler/container-disk/mount_test.go":                           nil,
				"pkg/virt-handler/vm_test.go":                                             nil,
				"pkg/container-disk/container-disk.go":                                    nil,
			}, ""),
		},
		{
			name: "kubevirtci-bump with deleted files",
			fields: fields{
				relevantFileDiffs: []*diff.FileDiff{
					diffFilePathsToDiffs["testdata/kubevirt/ci-bump-remove-provider-local/bump-kci.patch00"],
					diffFilePathsToDiffs["testdata/kubevirt/ci-bump-remove-provider-local/bump-kci.patch01"],
					diffFilePathsToDiffs["testdata/kubevirt/ci-bump-remove-provider-local/bump-kci.patch02"],
					diffFilePathsToDiffs["testdata/kubevirt/ci-bump-remove-provider-local/bump-kci.patch03"],
					diffFilePathsToDiffs["testdata/kubevirt/ci-bump-remove-provider-local/bump-kci.patch04"],
					diffFilePathsToDiffs["testdata/kubevirt/ci-bump-remove-provider-local/bump-kci.patch05"],
				},
			},
			want: newReviewResultWithData(bumpKubevirtCIApproveComment, bumpKubevirtCIDisapproveComment, nil, ""),
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

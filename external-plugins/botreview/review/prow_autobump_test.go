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

func TestProwAutobump_Review(t1 *testing.T) {
	diffFilePathes := []string{}
	entries, err := os.ReadDir("testdata/prow-autobump")
	if err != nil {
		t1.Errorf("failed to read files: %v", err)
	}
	for _, entry := range entries {
		diffFilePathes = append(diffFilePathes, filepath.Join("testdata/prow-autobump", entry.Name()))
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
		want   *ProwAutobumpResult
	}{
		{
			name: "simple prow autobump",
			fields: fields{
				relevantFileDiffs: []*diff.FileDiff{
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_configs_current_config_config.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_local_branch-protector.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_local_cherrypicker_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_local_label-sync-kubevirt.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_local_label-sync-nmstate.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_crier_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_deck_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_ghproxy.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_hook_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_horologium_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_needs-rebase_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_prow_controller_manager_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_sinker_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_statusreconciler_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_tide_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_overlays_ibmcloud-production_resources_prow-exporter-deployment.yaml"],
				},
			},
			want: &ProwAutobumpResult{},
		},
		{
			name: "prow autobump with crd update",
			fields: fields{
				relevantFileDiffs: []*diff.FileDiff{
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_configs_current_config_config.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_local_branch-protector.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_local_cherrypicker_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_local_label-sync-kubevirt.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_local_label-sync-nmstate.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_crier_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_deck_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_ghproxy.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_hook_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_horologium_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_needs-rebase_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_prow_controller_manager_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_prowjob-crd_prowjob_customresourcedefinition.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_sinker_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_statusreconciler_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_tide_deployment.yaml"],
					diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_overlays_ibmcloud-production_resources_prow-exporter-deployment.yaml"],
				},
			},
			want: &ProwAutobumpResult{
				notMatchingHunks: map[string][]*diff.Hunk{
					"github/ci/prow-deploy/kustom/base/manifests/test_infra/current/prowjob-crd/prowjob_customresourcedefinition.yaml": diffFilePathesToDiffs["testdata/prow-autobump/github_ci_prow-deploy_kustom_base_manifests_test_infra_current_prowjob-crd_prowjob_customresourcedefinition.yaml"].Hunks,
				},
			},
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &ProwAutobump{
				relevantFileDiffs: tt.fields.relevantFileDiffs,
			}
			if got := t.Review(); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("Review() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
 *
 */

package imagebump

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

var kubevirtCIImageRepo = regexp.MustCompile(`quay\.io/kubevirtci/([^:\s@"']+)`)

// KubevirtCIRepoNames scans github/ci/prow-deploy for quay.io/kubevirtci/<repo> references
// (image repository names only), matching hack/_include_image_funcs.sh kubevirtci_images_used_in_manifests.
func KubevirtCIRepoNames(repoRoot string) ([]string, error) {
	prowDeploy := filepath.Join(repoRoot, "github/ci/prow-deploy")
	names := map[string]struct{}{}
	err := filepath.WalkDir(prowDeploy, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !stringsHasYAMLExt(path) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range kubevirtCIImageRepo.FindAllSubmatch(data, -1) {
			names[string(m[1])] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(names))
	for n := range names {
		out = append(out, n)
	}
	sort.Strings(out)
	return out, nil
}

func stringsHasYAMLExt(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".yaml" || ext == ".yml"
}

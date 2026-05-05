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
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

func requireSkopeo() error {
	if _, err := exec.LookPath("skopeo"); err != nil {
		return fmt.Errorf("skopeo is required for this command: %w", err)
	}
	return nil
}

const kubevirtCIPrefix = "quay.io/kubevirtci/"

func imageRepoName(imageRef string) string {
	if !strings.HasPrefix(imageRef, kubevirtCIPrefix) {
		return imageRef
	}
	return imageRef[len(kubevirtCIPrefix):]
}

var deploymentRelDirs = []string{
	"github/ci/prow-deploy/kustom/base/manifests/local",
	"github/ci/prow-deploy/kustom/overlays/prow-workloads/resources",
}

// ResolveKubevirtCITagMap returns quay.io/kubevirtci/<repo> -> latest matching tag
// for each repository discovered under github/ci/prow-deploy.
func ResolveKubevirtCITagMap(repoRoot string) (map[string]string, error) {
	if err := requireSkopeo(); err != nil {
		return nil, err
	}
	repos, err := KubevirtCIRepoNames(repoRoot)
	if err != nil {
		return nil, err
	}
	m := make(map[string]string, len(repos))
	for _, repoName := range repos {
		imageRef := kubevirtCIPrefix + repoName
		tag, err := LatestSkopeoTag(imageRef)
		if err != nil {
			if errors.Is(err, ErrNoMatchingTag) {
				logrus.WithError(err).Warn("skipping image bump for repository with no matching tags")
				continue
			}
			return nil, fmt.Errorf("%s: %w", imageRef, err)
		}
		m[imageRef] = tag
	}
	return m, nil
}

// BumpJobImagesWithTagMap applies the given tag map to prow job configs. It
// matches BumpJobImages semantics (bootstrap-legacy is skipped in job YAML).
func BumpJobImagesWithTagMap(repoRoot string, imageToTag map[string]string) error {
	jobDir := filepath.Join(repoRoot, "github/ci/prow-deploy/files/jobs")
	for _, imageRef := range sortedImageRefs(imageToTag) {
		if imageRepoName(imageRef) == "bootstrap-legacy" {
			continue
		}
		tag := imageToTag[imageRef]
		if err := replaceInJobYAMLs(jobDir, imageRef, tag); err != nil {
			return err
		}
	}
	return nil
}

// BumpProwDeploymentImagesWithTagMap updates kustom deployment YAML for each
// quay.io/kubevirtci/ image in the map.
func BumpProwDeploymentImagesWithTagMap(repoRoot string, imageToTag map[string]string) error {
	for _, imageRef := range sortedImageRefs(imageToTag) {
		tag := imageToTag[imageRef]
		for _, rel := range deploymentRelDirs {
			dir := filepath.Join(repoRoot, rel)
			if err := replaceInYAMLTree(dir, imageRef, tag); err != nil {
				return err
			}
		}
	}
	return nil
}

func sortedImageRefs(imageToTag map[string]string) []string {
	refs := make([]string, 0, len(imageToTag))
	for r := range imageToTag {
		refs = append(refs, r)
	}
	sort.Strings(refs)
	return refs
}

// BumpJobImages updates kubevirtci image references under prow job configs.
func BumpJobImages(repoRoot string) error {
	m, err := ResolveKubevirtCITagMap(repoRoot)
	if err != nil {
		return err
	}
	return BumpJobImagesWithTagMap(repoRoot, m)
}

func replaceInJobYAMLs(jobDir, imageRef, newTag string) error {
	return filepath.WalkDir(jobDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !IsJobConfigPath(path) {
			return nil
		}
		return replaceInFile(path, func(body string) string {
			return ReplaceKubevirtCIImage(body, imageRef, newTag)
		})
	})
}

// BumpProwDeploymentImages updates kubevirtci image references in prow kustom manifests.
func BumpProwDeploymentImages(repoRoot string) error {
	m, err := ResolveKubevirtCITagMap(repoRoot)
	if err != nil {
		return err
	}
	return BumpProwDeploymentImagesWithTagMap(repoRoot, m)
}

func replaceInYAMLTree(dir, imageRef, newTag string) error {
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".yaml" && filepath.Ext(path) != ".yml" {
			return nil
		}
		return replaceInFile(path, func(body string) string {
			return ReplaceKubevirtCIImage(body, imageRef, newTag)
		})
	})
}

func replaceInFile(path string, transform func(string) string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	mode := info.Mode() & os.ModePerm
	orig := string(data)
	next := transform(orig)
	if orig == next {
		return nil
	}
	return os.WriteFile(path, []byte(next), mode)
}

// BumpContainerfileImages updates FROM lines for quay.io images using the quay API.
func BumpContainerfileImages(repoRoot string) error {
	paths, err := gitLsFiles(repoRoot, func(name string) bool {
		return strings.HasSuffix(name, "Containerfile") || strings.HasSuffix(name, "Dockerfile")
	})
	if err != nil {
		return err
	}
	for _, rel := range paths {
		path := filepath.Join(repoRoot, rel)
		if err := bumpOneContainerfile(path); err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
	}
	return nil
}

func gitLsFiles(repoRoot string, keep func(string) bool) ([]string, error) {
	cmd := exec.Command("git", "-C", repoRoot, "ls-files")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files: %w", err)
	}
	var paths []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !keep(line) {
			continue
		}
		paths = append(paths, line)
	}
	return paths, nil
}

func bumpOneContainerfile(path string) error {
	lines, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(lines)
	endsWithNL := len(text) > 0 && text[len(text)-1] == '\n'
	parts := strings.Split(text, "\n")
	changed := false
	for i, line := range parts {
		img, ok := fromLineImage(line)
		if !ok {
			continue
		}
		repo, oldTag, ok := SplitImageRef(img)
		if !ok {
			continue
		}
		latest, err := LatestQuayTag(repo)
		if err != nil {
			// Match bash: skip when quay has no tag (non-quay images, etc.)
			continue
		}
		if oldTag == latest {
			continue
		}
		oldFull := repo + ":" + oldTag
		newFull := repo + ":" + latest
		if strings.Contains(parts[i], oldFull) {
			parts[i] = strings.Replace(parts[i], oldFull, newFull, 1)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	out := strings.Join(parts, "\n")
	if endsWithNL && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	mode := info.Mode() & os.ModePerm
	return os.WriteFile(path, []byte(out), mode)
}

func fromLineImage(line string) (string, bool) {
	t := strings.TrimSpace(line)
	if len(t) < 6 || !strings.EqualFold(t[:4], "from") {
		return "", false
	}
	if t[4] != ' ' && t[4] != '\t' {
		return "", false
	}
	rest := strings.TrimSpace(t[5:])
	fields := strings.Fields(rest)
	idx := 0
	for idx < len(fields) && strings.HasPrefix(fields[idx], "--") {
		idx++
	}
	if idx >= len(fields) {
		return "", false
	}
	var b strings.Builder
	for ; idx < len(fields); idx++ {
		if strings.EqualFold(fields[idx], "AS") {
			break
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(fields[idx])
	}
	s := b.String()
	if s == "" {
		return "", false
	}
	return s, true
}

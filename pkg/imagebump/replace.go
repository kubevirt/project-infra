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
	"regexp"
	"strings"
)

var imageRefSuffix = regexp.MustCompile(`(@sha256:[^\n]*|:v?[a-z0-9]+-[^\n]*)`)

// ReplaceKubevirtCIImage updates content so imageRef (e.g. quay.io/kubevirtci/golang) is
// pinned to newTag (without leading ':').
func ReplaceKubevirtCIImage(content, imageRef, newTag string) string {
	re := regexp.MustCompile(regexp.QuoteMeta(imageRef) + imageRefSuffix.String())
	return re.ReplaceAllString(content, imageRef+":"+newTag)
}

// JobConfigPath matches prow job YAML paths used by update-jobs-with-latest-image.sh.
var JobConfigPath = regexp.MustCompile(`.*-(?:periodics|presubmits|postsubmits)(?:-master|-main)?\.yaml$`)

// IsJobConfigPath returns whether path should receive prow job image bumps.
func IsJobConfigPath(path string) bool {
	return JobConfigPath.MatchString(path)
}

// SplitImageRef splits "repo:tag" into repo and tag.
func SplitImageRef(image string) (repo, tag string, ok bool) {
	if strings.Contains(image, "@sha256:") {
		return "", "", false
	}
	i := strings.LastIndex(image, ":")
	if i <= 0 {
		return "", "", false
	}
	// Ignore IPv6 host brackets or port-like segments in host (no tag)
	if strings.Contains(image[i+1:], "/") {
		return "", "", false
	}
	return image[:i], image[i+1:], true
}

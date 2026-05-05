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
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
)

// kubevirtCITagPattern matches tags selected by hack/update-jobs-with-latest-image.sh
var kubevirtCITagPattern = regexp.MustCompile(`^v?[0-9]+-[a-z0-9]{7,9}$`)
var ErrNoMatchingTag = errors.New("no tag matching kubevirtci date-hash pattern")

type skopeoListTags struct {
	Tags []string `json:"Tags"`
}

// LatestSkopeoTag returns the last tag in skopeo list-tags order that matches kubevirtCITagPattern
func LatestSkopeoTag(imageRef string) (string, error) {
	cmd := exec.Command("skopeo", "list-tags", "docker://"+imageRef)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("skopeo list-tags docker://%s: %w\n%s", imageRef, err, string(ee.Stderr))
		}
		return "", fmt.Errorf("skopeo list-tags docker://%s: %w", imageRef, err)
	}
	var parsed skopeoListTags
	if err := json.Unmarshal(out, &parsed); err != nil {
		return "", fmt.Errorf("parse skopeo output: %w", err)
	}
	var last string
	for _, t := range parsed.Tags {
		if kubevirtCITagPattern.MatchString(t) {
			last = t
		}
	}
	if last == "" {
		return "", fmt.Errorf("%w for %s (pattern %s)", ErrNoMatchingTag, imageRef, kubevirtCITagPattern.String())
	}
	return last, nil
}

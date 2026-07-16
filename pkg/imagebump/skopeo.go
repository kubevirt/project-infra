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
	"sort"
	"strings"
	"time"
)

// kubevirtCITagPattern matches tags selected by hack/update-jobs-with-latest-image.sh
var kubevirtCITagPattern = regexp.MustCompile(`^v?[0-9]+-[a-z0-9]{7,9}$`)
var ErrNoMatchingTag = errors.New("no tag matching kubevirtci date-hash pattern")

type listTagsCommand func(imageRef string) ([]string, error)

var listTagsCmd = func(imageRef string) ([]string, error) {
	cmd := exec.Command("skopeo", "list-tags", "docker://"+imageRef)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("skopeo list-tags docker://%s: %w\n%s", imageRef, err, string(ee.Stderr))
		}
		return nil, fmt.Errorf("skopeo list-tags docker://%s: %w", imageRef, err)
	}
	type skopeoListTags struct {
		Tags []string `json:"Tags"`
	}
	var parsed skopeoListTags
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("parse skopeo output: %w", err)
	}
	return parsed.Tags, nil
}

type inspectCommand func(fullImageRef string) (time.Time, error)

var inspectCmd = func(fullImageRef string) (time.Time, error) {
	cmd := exec.Command("skopeo", "inspect", "docker://"+fullImageRef)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return time.Time{}, fmt.Errorf("skopeo inspect docker://%s: %w\n%s", fullImageRef, err, string(ee.Stderr))
		}
		return time.Time{}, fmt.Errorf("skopeo inspect docker://%s: %w", fullImageRef, err)
	}
	type skopeoInspect struct {
		Created time.Time `json:"Created"`
	}
	var parsed skopeoInspect
	if err := json.Unmarshal(out, &parsed); err != nil {
		return time.Time{}, fmt.Errorf("parse skopeo inspect output: %w", err)
	}
	return parsed.Created, nil
}

// LatestSkopeoTag returns the last tag in skopeo list-tags order that matches kubevirtCITagPattern
// after sorting by the date part first - if that doesn't work, i.e. if the date part is the same for
// two tags, then it resorts to calling skopeo inspect, and comparing the Created entry
func LatestSkopeoTag(imageRef string) (string, error) {
	tags, err := listTagsCmd(imageRef)
	if err != nil {
		return "", fmt.Errorf("listTagsCmd for %s: %w", imageRef, err)
	}
	var allMatching []string
	for _, t := range tags {
		if kubevirtCITagPattern.MatchString(t) {
			allMatching = append(allMatching, t)
		}
	}
	if len(allMatching) == 0 {
		return "", fmt.Errorf("%w for %s (pattern %s)", ErrNoMatchingTag, imageRef, kubevirtCITagPattern.String())
	}

	var sortErrs []error
	sort.Slice(allMatching, func(i, j int) bool {
		iDateTime, jDateTime := strings.Split(allMatching[i], "-")[0], strings.Split(allMatching[j], "-")[0]
		if iDateTime != jDateTime {
			return iDateTime < jDateTime
		}
		iCreated, err := inspectCmd(fmt.Sprintf("%s:%s", imageRef, allMatching[i]))
		if err != nil {
			sortErrs = append(sortErrs, err)
			return false
		}
		jCreated, err := inspectCmd(fmt.Sprintf("%s:%s", imageRef, allMatching[j]))
		if err != nil {
			sortErrs = append(sortErrs, err)
			return false
		}
		return iCreated.Before(jCreated)
	})
	if len(sortErrs) > 0 {
		return "", fmt.Errorf("errors when sorting tags: %w", errors.Join(sortErrs...))
	}

	return allMatching[len(allMatching)-1], nil
}

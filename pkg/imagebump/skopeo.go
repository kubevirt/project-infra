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
	"slices"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

// kubevirtCITagPattern matches tags selected by hack/update-jobs-with-latest-image.sh
var kubevirtCITagPattern = regexp.MustCompile(`^v?[0-9]+-[a-z0-9]{7,9}$`)
var ErrNoMatchingTag = errors.New("no tag matching kubevirtci date-hash pattern")

type listTagsCommand func(imageRef string) ([]string, error)

var listTagsCmd = func(imageRef string) ([]string, error) {
	cmdArgs := []string{"list-tags", "docker://" + imageRef}
	fullCommand := "skopeo " + strings.Join(cmdArgs, " ")
	logEntry := log.WithField("command", fullCommand)
	logEntry.Debug("running")
	cmd := exec.Command("skopeo", cmdArgs...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%s: %w\n%s", fullCommand, err, string(ee.Stderr))
		}
		return nil, fmt.Errorf("%s: %w", fullCommand, err)
	}
	type skopeoListTags struct {
		Tags []string `json:"Tags"`
	}
	var parsed skopeoListTags
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("parse %s output: %w", fullCommand, err)
	}
	logEntry.WithField("parsed", parsed).Debug("done")
	return parsed.Tags, nil
}

type inspectCommand func(fullImageRef string) (time.Time, error)

var inspectCmd = func(fullImageRef string) (time.Time, error) {
	const dateTimeFormat string = "2006-01-02T15:04:05Z"
	cmdArgs := []string{"inspect", "--no-tags", fmt.Sprintf(`--format={{.Created.Format "%s"}}`, dateTimeFormat), "docker://" + fullImageRef}
	fullCommand := "skopeo " + strings.Join(cmdArgs, " ")
	logEntry := log.WithField("command", fullCommand)
	logEntry.Debug("running")
	cmd := exec.Command("skopeo", cmdArgs...)
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return time.Time{}, fmt.Errorf("%s: %w\n%s", fullCommand, err, string(ee.Stderr))
		}
		return time.Time{}, fmt.Errorf("%s: %w", fullCommand, err)
	}
	before, _ := strings.CutSuffix(string(out), "\n")
	parsed, err := time.Parse(dateTimeFormat, before)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse %s output: %w\n%v", fullCommand, err, out)
	}
	logEntry.WithField("parsed", parsed).Debug("done")
	return parsed, nil
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

	// First pass, reverse sort, so newest elements should be at the start
	slices.Reverse(allMatching)

	// Now reduce to the subset of the latest candidates, i.e. remove all that don't have the date component of the last
	// entry
	dateComponent := strings.Split(allMatching[0], "-")[0]
	var candidates []string
	for _, c := range allMatching {
		if !strings.HasPrefix(c, dateComponent) {
			continue
		}
		candidates = append(candidates, c)
	}

	// Only one left, done
	if len(candidates) == 1 {
		return candidates[0], nil
	}

	// Now sort the candidates again, using skopeo inspect to look up the Created date
	// This time the most recent tag will show up at the bottom
	var sortErrs []error
	sort.Slice(candidates, func(i, j int) bool {
		iCreated, err := inspectCmd(fmt.Sprintf("%s:%s", imageRef, candidates[i]))
		if err != nil {
			sortErrs = append(sortErrs, err)
			return false
		}
		jCreated, err := inspectCmd(fmt.Sprintf("%s:%s", imageRef, candidates[j]))
		if err != nil {
			sortErrs = append(sortErrs, err)
			return false
		}
		return iCreated.Before(jCreated)
	})
	if len(sortErrs) > 0 {
		return "", fmt.Errorf("errors when sorting tags: %w", errors.Join(sortErrs...))
	}

	return allMatching[len(candidates)-1], nil
}

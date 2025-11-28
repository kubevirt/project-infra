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

package github

import (
	"fmt"
	"strings"
)

// OrgRepo extracts org and repo from a string in format "{org}/{repo}". It returns an error if the input is not in the format described
func OrgRepo(eventRepoFullName string) (string, string, error) {
	orgRepo := strings.Split(eventRepoFullName, "/")
	org, repo := orgRepo[0], orgRepo[1]
	if len(orgRepo) != 2 || org == "" || repo == "" {
		return "", "", fmt.Errorf("input %q has unexpected format, should be \"{org}/{repo}\"", eventRepoFullName)
	}
	return org, repo, nil
}

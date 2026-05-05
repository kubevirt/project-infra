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
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type quayTagList struct {
	Tags []struct {
		Name string `json:"name"`
	} `json:"tags"`
}

// LatestQuayTag returns the first tag from quay.io/api/v1/repository/.../tag/?limit=1
// that is not "latest", matching hack/_include_image_funcs.sh latest_image_tag.
func LatestQuayTag(fullImage string) (string, error) {
	if !strings.HasPrefix(fullImage, "quay.io/") {
		return "", fmt.Errorf("expected quay.io image, got %q", fullImage)
	}
	repoPath := strings.TrimPrefix(fullImage, "quay.io/")
	url := "https://quay.io/api/v1/repository/" + repoPath + "/tag/?limit=1"
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("quay API %s: %s", resp.Status, string(body))
	}
	var parsed quayTagList
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", err
	}
	for _, t := range parsed.Tags {
		if t.Name != "" && t.Name != "latest" {
			return t.Name, nil
		}
	}
	return "", fmt.Errorf("no usable tag in quay response for %s", fullImage)
}

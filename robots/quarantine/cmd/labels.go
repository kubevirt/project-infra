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

package cmd

import (
	"fmt"
	"os"
	"strings"

	"sigs.k8s.io/yaml"
)

type labelsConfig struct {
	Default struct {
		Labels []labelEntry `json:"labels"`
	} `json:"default"`
}

type labelEntry struct {
	Name string `json:"name"`
}

// ginkgoLabelAliases maps Ginkgo test labels that use different naming than
// the corresponding Prow labels defined in labels.yaml. The value is the Prow
// command string (e.g. "sig observability", "wg arch-s390x").
var ginkgoLabelAliases = map[string]string{
	"sig-monitoring":  "sig observability",
	"sig-performance": "sig scale",
	"wg-s390x":        "wg arch-s390x",
	"wg-arm64":        "wg arch-arm",
}

func loadValidGroupsFromLabelsFile(path string) (sigs, wgs map[string]bool, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("could not read labels file %q: %w", path, err)
	}
	var cfg labelsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, nil, fmt.Errorf("could not parse labels file %q: %w", path, err)
	}
	sigs = make(map[string]bool)
	wgs = make(map[string]bool)
	for _, entry := range cfg.Default.Labels {
		if name, ok := strings.CutPrefix(entry.Name, "sig/"); ok {
			sigs[name] = true
		} else if name, ok := strings.CutPrefix(entry.Name, "wg/"); ok {
			wgs[name] = true
		}
	}
	if len(sigs) == 0 {
		return nil, nil, fmt.Errorf("no SIG labels found in %q", path)
	}
	return sigs, wgs, nil
}

const defaultProwCommand = "sig compute"

// resolveProwCommand maps a Ginkgo test label (e.g. "sig-compute-migrations")
// to a Prow command string (e.g. "sig compute") using the following resolution:
//  1. Alias: known renames (sig-monitoring → sig observability)
//  2. Direct match: label suffix matches a valid SIG or WG
//  3. Prefix match: progressively shorter prefixes of the label suffix
//  4. Fallback: "sig compute"
func resolveProwCommand(ginkgoLabel string, validSIGs, validWGs map[string]bool) string {
	if cmd, ok := ginkgoLabelAliases[ginkgoLabel]; ok {
		return cmd
	}

	if suffix, ok := strings.CutPrefix(ginkgoLabel, "sig-"); ok {
		if validSIGs[suffix] {
			return "sig " + suffix
		}
		for i := len(suffix) - 1; i >= 0; i-- {
			if suffix[i] == '-' {
				prefix := suffix[:i]
				if validSIGs[prefix] {
					return "sig " + prefix
				}
			}
		}
	} else if suffix, ok := strings.CutPrefix(ginkgoLabel, "wg-"); ok {
		if validWGs[suffix] {
			return "wg " + suffix
		}
		for i := len(suffix) - 1; i >= 0; i-- {
			if suffix[i] == '-' {
				prefix := suffix[:i]
				if validWGs[prefix] {
					return "wg " + prefix
				}
			}
		}
	}

	return defaultProwCommand
}

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

package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadJobNamesFromConfig(t *testing.T) {
	tests := []struct {
		name           string
		yamlContent    string
		expectedNames  []string
		expectError    bool
	}{
		{
			name: "selects latest k8s version per sig",
			yamlContent: `periodics:
- name: periodic-kubevirt-e2e-k8s-1.33-sig-compute
  cron: "0 0 * * *"
  spec:
    containers:
    - image: test
- name: periodic-kubevirt-e2e-k8s-1.35-sig-compute
  cron: "0 0 * * *"
  spec:
    containers:
    - image: test
- name: periodic-kubevirt-e2e-k8s-1.34-sig-storage
  cron: "0 0 * * *"
  spec:
    containers:
    - image: test
- name: periodic-kubevirt-e2e-k8s-1.35-sig-storage
  cron: "0 0 * * *"
  spec:
    containers:
    - image: test
- name: periodic-kubevirt-e2e-k8s-1.35-sig-network
  cron: "0 0 * * *"
  spec:
    containers:
    - image: test
- name: periodic-kubevirt-e2e-k8s-1.35-sig-operator
  cron: "0 0 * * *"
  spec:
    containers:
    - image: test
`,
			expectedNames: []string{
				"periodic-kubevirt-e2e-k8s-1.35-sig-compute",
				"periodic-kubevirt-e2e-k8s-1.35-sig-network",
				"periodic-kubevirt-e2e-k8s-1.35-sig-operator",
				"periodic-kubevirt-e2e-k8s-1.35-sig-storage",
			},
		},
		{
			name: "ignores non-main-sig jobs",
			yamlContent: `periodics:
- name: periodic-kubevirt-e2e-k8s-1.35-sig-compute
  cron: "0 0 * * *"
  spec:
    containers:
    - image: test
- name: periodic-kubevirt-e2e-k8s-1.35-sig-compute-migrations
  cron: "0 0 * * *"
  spec:
    containers:
    - image: test
- name: periodic-kubevirt-e2e-k8s-1.35-sig-compute-root
  cron: "0 0 * * *"
  spec:
    containers:
    - image: test
- name: periodic-kubevirt-e2e-k8s-1.35-sig-performance
  cron: "0 0 * * *"
  spec:
    containers:
    - image: test
- name: periodic-kubevirt-push-test-metrics
  cron: "0 0 * * *"
  spec:
    containers:
    - image: test
- name: periodic-kubevirt-e2e-k8s-1.35-ipv6-sig-network
  cron: "0 0 * * *"
  spec:
    containers:
    - image: test
`,
			expectedNames: []string{
				"periodic-kubevirt-e2e-k8s-1.35-sig-compute",
			},
		},
		{
			name: "error on no matching jobs",
			yamlContent: `periodics:
- name: periodic-kubevirt-push-test-metrics
  cron: "0 0 * * *"
  spec:
    containers:
    - image: test
`,
			expectError: true,
		},
		{
			name: "handles multiple k8s versions correctly",
			yamlContent: `periodics:
- name: periodic-kubevirt-e2e-k8s-1.33-sig-network
  cron: "0 0 * * *"
  spec:
    containers:
    - image: test
- name: periodic-kubevirt-e2e-k8s-1.34-sig-network
  cron: "0 0 * * *"
  spec:
    containers:
    - image: test
- name: periodic-kubevirt-e2e-k8s-1.35-sig-network
  cron: "0 0 * * *"
  spec:
    containers:
    - image: test
`,
			expectedNames: []string{
				"periodic-kubevirt-e2e-k8s-1.35-sig-network",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "periodics.yaml")
			if err := os.WriteFile(configFile, []byte(tc.yamlContent), 0644); err != nil {
				t.Fatalf("failed to write temp config: %v", err)
			}

			names, err := readJobNamesFromConfig(configFile)
			if tc.expectError {
				if err == nil {
					t.Fatal("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(names) != len(tc.expectedNames) {
				t.Fatalf("expected %d job names, got %d: %v", len(tc.expectedNames), len(names), names)
			}
			for i, expected := range tc.expectedNames {
				if names[i] != expected {
					t.Errorf("job name[%d]: expected %q, got %q", i, expected, names[i])
				}
			}
		})
	}
}

func TestReadJobNamesFromConfig_invalidPath(t *testing.T) {
	_, err := readJobNamesFromConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent config path")
	}
}

func TestJobNamesFlag(t *testing.T) {
	var names jobNames
	if err := names.Set("job-a"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := names.Set("job-b"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %d", len(names))
	}
	if names[0] != "job-a" || names[1] != "job-b" {
		t.Errorf("unexpected names: %v", names)
	}
	if names.String() != "job-a,job-b" {
		t.Errorf("unexpected String() output: %q", names.String())
	}
}

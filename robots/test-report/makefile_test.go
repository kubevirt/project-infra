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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPushWithoutGo(t *testing.T) {
	makePath, err := exec.LookPath("make")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name           string
		publishExit    string
		registryExit   string
		wantErr        bool
		wantScriptBump bool
	}{
		{"success", "0", "0", false, true},
		{"publish failure", "42", "0", true, false},
		{"script update failure", "0", "42", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := t.TempDir()
			for _, rel := range []string{
				"robots/test-report/Makefile",
				"hack/update-jobs-with-latest-image.sh",
				"hack/update-scripts-with-latest-image.sh",
				"hack/test-report.sh",
			} {
				data, err := os.ReadFile(filepath.Join("../..", rel))
				if err != nil {
					t.Fatal(err)
				}
				writePushFixture(t, filepath.Join(fixture, rel), string(data))
			}
			if err := os.MkdirAll(filepath.Join(fixture, "github/ci/prow-deploy/files/jobs"), 0755); err != nil {
				t.Fatal(err)
			}
			binDir := filepath.Join(fixture, "bin")
			if err := os.MkdirAll(binDir, 0755); err != nil {
				t.Fatal(err)
			}
			// Only expose the shell helpers, never the host Go toolchain.
			for _, tool := range []string{"bash", "dirname", "readlink", "jq", "tail", "find", "sed"} {
				path, err := exec.LookPath(tool)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path, filepath.Join(binDir, tool)); err != nil {
					t.Fatal(err)
				}
			}
			writePushFixture(t, filepath.Join(fixture, "images/publish_image.sh"), `#!/bin/bash
printf 'publish %s\n' "$*" >> "$CALLS"
exit "$PUBLISH_EXIT"
`)
			writePushFixture(t, filepath.Join(binDir, "skopeo"), `#!/bin/bash
printf 'skopeo %s\n' "$*" >> "$CALLS"
if [ "$REGISTRY_EXIT" != 0 ]; then
    exit "$REGISTRY_EXIT"
fi
printf '%s\n' '{"Tags":["v209901010000-abcdef0"]}'
`)
			calls := filepath.Join(fixture, "calls.log")
			command := exec.Command(makePath, "--no-print-directory", "push")
			command.Dir = filepath.Join(fixture, "robots/test-report")
			command.Env = append(os.Environ(), "PATH="+binDir, "MAKEFLAGS=", "MFLAGS=",
				"CALLS="+calls, "PUBLISH_EXIT="+tc.publishExit, "REGISTRY_EXIT="+tc.registryExit)
			output, err := command.CombinedOutput()
			if (err != nil) != tc.wantErr {
				t.Fatalf("make push error = %v, wantErr = %v\n%s", err, tc.wantErr, output)
			}
			if strings.Contains(string(output), "go: command not found") {
				t.Fatalf("make push must not require Go\n%s", output)
			}
			gotCalls, err := os.ReadFile(calls)
			if err != nil {
				t.Fatal(err)
			}
			wantCalls := "publish -e test-report quay.io kubevirtci\n"
			if tc.publishExit == "0" {
				wantCalls += "skopeo list-tags docker://quay.io/kubevirtci/test-report\n"
			}
			if string(gotCalls) != wantCalls {
				t.Fatalf("calls = %q, want %q", gotCalls, wantCalls)
			}
			script, err := os.ReadFile(filepath.Join(fixture, "hack/test-report.sh"))
			if err != nil {
				t.Fatal(err)
			}
			if bumped := strings.Contains(string(script), "quay.io/kubevirtci/test-report:v209901010000-abcdef0"); bumped != tc.wantScriptBump {
				t.Fatalf("script image bumped = %v, want %v", bumped, tc.wantScriptBump)
			}
		})
	}
}

func writePushFixture(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0755); err != nil {
		t.Fatal(err)
	}
}

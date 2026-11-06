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
 */

package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const fakeSkopeo = `#!/bin/sh
case "$1" in
    list-tags) printf '%s\n' "$TEST_IMAGE_TAGS" ;;
    inspect) printf '%s\n' '2026-11-05T09:30:00Z' ;;
    *) exit 1 ;;
esac
`

func TestLatestTagCommand(t *testing.T) {
	binDir := t.TempDir()
	writeTestFile(t, filepath.Join(binDir, "skopeo"), fakeSkopeo, 0755)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, tc := range []struct {
		name    string
		args    []string
		tags    string
		want    string
		wantErr bool
	}{
		{"mixed prefixes", []string{"quay.io/kubevirtci/phased"}, `{"Tags":["202611050920-2a81b11","v202607231917-ca75c4e"]}`, "202611050920-2a81b11\n", false},
		{"no matching tag", []string{"quay.io/kubevirtci/phased"}, `{"Tags":["latest"]}`, "", true},
		{"registry error", []string{"quay.io/kubevirtci/phased"}, `not json`, "", true},
		{"missing image", nil, "", "", true},
		{"extra argument", []string{"one", "two"}, "", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TEST_IMAGE_TAGS", tc.tags)
			cmd := newLatestTagCommand()
			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			cmd.SilenceUsage = true
			cmd.SetArgs(tc.args)
			err := cmd.Execute()
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr = %v", err, tc.wantErr)
			}
			if stdout.String() != tc.want {
				t.Fatalf("stdout = %q, want %q", stdout.String(), tc.want)
			}
		})
	}
}

func TestExecuteRegistersLatestTagCommand(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	os.Args = []string{"image-bumper", "latest-tag"}
	err := Execute()
	if err == nil || !strings.Contains(err.Error(), "accepts 1 arg") {
		t.Fatalf("expected latest-tag argument validation, got %v", err)
	}
}

func TestManualImageUpdateScripts(t *testing.T) {
	repoRoot, err := filepath.Abs("../../..")
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "image-bumper")
	build := exec.Command("go", "build", "-o", binary, "./robots/image-bumper")
	build.Dir = repoRoot
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build image-bumper: %v\n%s", err, output)
	}
	const oldImage = "quay.io/kubevirtci/phased:v202607231917-ca75c4e"
	const newImage = "quay.io/kubevirtci/phased:202611050920-2a81b11"
	for _, tc := range []struct {
		name     string
		script   string
		explicit bool
		noTags   bool
	}{
		{"jobs", "update-jobs-with-latest-image.sh", false, false},
		{"deployments", "update-deployments-with-latest-image.sh", false, false},
		{"explicit deployment tag", "update-deployments-with-latest-image.sh", true, true},
		{"jobs without matching tags", "update-jobs-with-latest-image.sh", false, true},
		{"deployments without matching tags", "update-deployments-with-latest-image.sh", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fixture := t.TempDir()
			scriptPath := filepath.Join(fixture, "hack", tc.script)
			script, err := os.ReadFile(filepath.Join(repoRoot, "hack", tc.script))
			if err != nil {
				t.Fatal(err)
			}
			writeTestFile(t, scriptPath, string(script), 0755)
			binDir := filepath.Join(fixture, "bin")
			writeTestFile(t, filepath.Join(binDir, "skopeo"), fakeSkopeo, 0755)
			// Run the real selector while verifying the scripts' go-run invocation.
			writeTestFile(t, filepath.Join(binDir, "go"), `#!/bin/sh
test "$1" = run || exit 1
test "$2" = ./robots/image-bumper || exit 1
test "$3" = latest-tag || exit 1
shift 2
exec "$TEST_IMAGE_BUMPER" "$@"
`, 0755)
			t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("TEST_IMAGE_BUMPER", binary)
			tags := `{"Tags":["202611050920-2a81b11","v202607231917-ca75c4e"]}`
			if tc.noTags {
				tags = `{"Tags":["latest"]}`
			}
			t.Setenv("TEST_IMAGE_TAGS", tags)
			paths := []string{
				"github/ci/prow-deploy/files/jobs/example-presubmits.yaml",
				"github/ci/prow-deploy/kustom/base/manifests/local/plugin.yaml",
				"github/ci/prow-deploy/kustom/overlays/prow-workloads/resources/plugin.yaml",
			}
			for _, path := range paths {
				writeTestFile(t, filepath.Join(fixture, path), "image: "+oldImage+"\n", 0644)
			}
			args := []string{scriptPath, "quay.io/kubevirtci/phased"}
			if tc.explicit {
				args = append(args, "202611050920-2a81b11")
			}
			command := exec.Command("bash", args...)
			command.Dir = t.TempDir() // Scripts must work from outside the checkout.
			output, err := command.CombinedOutput()
			wantErr := tc.noTags && !tc.explicit
			if (err != nil) != wantErr {
				t.Fatalf("script error = %v, wantErr = %v\n%s", err, wantErr, output)
			}
			for i, path := range paths {
				want := oldImage
				if !wantErr && (i == 0) == (tc.script == "update-jobs-with-latest-image.sh") {
					want = newImage
				}
				body, err := os.ReadFile(filepath.Join(fixture, path))
				if err != nil {
					t.Fatal(err)
				}
				if string(body) != "image: "+want+"\n" {
					t.Fatalf("%s contains %q, want image %s", path, body, want)
				}
			}
		})
	}
}

func writeTestFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

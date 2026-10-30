package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBazelFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		withWorkspace bool
		withModule    bool
		useWorkspace  bool
		useModule     bool
		wantBase      string
		wantErr       bool
	}{
		{
			name:         "falls_back_to_module_bazel_when_workspace_missing",
			withModule:   true,
			useWorkspace: true,
			wantBase:     "MODULE.bazel",
		},
		{
			name:          "prefers_workspace_when_present",
			withWorkspace: true,
			withModule:    true,
			useWorkspace:  true,
			wantBase:      "WORKSPACE",
		},
		{
			name:         "returns_workspace_path_when_both_missing",
			useWorkspace: true,
			wantBase:     "WORKSPACE",
		},
		{
			name:       "explicit_module_flag",
			withModule: true,
			useModule:  true,
			wantBase:   "MODULE.bazel",
		},
		{
			name:    "errors_when_neither_flag_set",
			wantErr: true,
		},
		{
			name:         "errors_when_both_flags_set",
			useWorkspace: true,
			useModule:    true,
			wantErr:      true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			wsPath := filepath.Join(dir, "WORKSPACE")
			modPath := filepath.Join(dir, "MODULE.bazel")
			if tc.withWorkspace {
				if err := os.WriteFile(wsPath, nil, 0644); err != nil {
					t.Fatal(err)
				}
			}
			if tc.withModule {
				if err := os.WriteFile(modPath, []byte("module(name = \"m\")\n"), 0644); err != nil {
					t.Fatal(err)
				}
			}

			var opts options
			if tc.useWorkspace {
				opts.workspacePath = wsPath
			}
			if tc.useModule {
				opts.modulePath = modPath
			}

			err := opts.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("opts.Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
			if !tc.wantErr {
				got := resolveBazelFile(opts)
				want := filepath.Join(dir, tc.wantBase)
				if got != want {
					t.Errorf("resolveBazelFile(%+v) = %q, want %q", opts, got, want)
				}
			}
		})
	}
}
